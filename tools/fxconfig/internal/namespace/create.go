/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package namespace

import (
	"context"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path"

	"github.com/hyperledger/fabric-lib-go/bccsp/sw"
	cb "github.com/hyperledger/fabric-protos-go-apiv2/common"
	ab "github.com/hyperledger/fabric-protos-go-apiv2/orderer"
	"github.com/hyperledger/fabric-x-common/api/applicationpb"
	"github.com/hyperledger/fabric-x-common/api/committerpb"
	"github.com/hyperledger/fabric-x-common/api/msppb"
	"github.com/hyperledger/fabric-x-common/cmd/common/comm"
	"github.com/hyperledger/fabric-x-common/msp"
	"github.com/hyperledger/fabric-x-common/protoutil"
	"github.com/hyperledger/fabric-x-common/protoutil/identity"
	"github.com/hyperledger/fabric-x-common/tools/configtxgen"
)

// DeployNamespace creates a namespace transactions and submits it to the ordering service.
func DeployNamespace(nsCfg NsConfig, ordererCfg OrdererConfig, mspCfg MSPConfig) error {
	err := validateConfig(nsCfg)
	if err != nil {
		return err
	}

	thisMSP, err := setupMSP(mspCfg)
	if err != nil {
		return fmt.Errorf("msp setup error: %w", err)
	}

	sid, err := thisMSP.GetDefaultSigningIdentity()
	if err != nil {
		return fmt.Errorf("get signer identity error: %w", err)
	}

	pkData, err := os.ReadFile(nsCfg.ThresholdPolicyVerificationKeyPath)
	if err != nil {
		return err
	}

	// we use the serialized public key as our namespace endorsement policy
	serializedPublicKey, err := getPubKeyFromPemData(pkData)
	if err != nil {
		return err
	}

	nsPolicy := &applicationpb.NamespacePolicy{
		Rule: &applicationpb.NamespacePolicy_ThresholdRule{
			ThresholdRule: &applicationpb.ThresholdRule{
				Scheme:    "ECDSA",
				PublicKey: serializedPublicKey,
			},
		},
	}

	tx := createNamespacesTx(nsPolicy, nsCfg.NamespaceID, nsCfg.Version)
	env, err := createSignedEnvelope(sid, nsCfg.Channel, tx)
	if err != nil {
		return err
	}

	return broadcast(ordererCfg, env)
}

// setupMSP instantiates a MSP based on the provided MSPConfig.
func setupMSP(mspCfg MSPConfig) (msp.MSP, error) { //nolint:ireturn
	conf, err := msp.GetLocalMspConfig(mspCfg.MSPConfigPath, nil, mspCfg.MSPID)
	if err != nil {
		return nil, fmt.Errorf("error getting local msp config from %v: %w", mspCfg.MSPConfigPath, err)
	}

	dir := path.Join(mspCfg.MSPConfigPath, "keystore")
	ks, err := sw.NewFileBasedKeyStore(nil, dir, true)
	if err != nil {
		return nil, err
	}

	cp, err := sw.NewDefaultSecurityLevelWithKeystore(ks)
	if err != nil {
		return nil, err
	}

	mspOpts := &msp.BCCSPNewOpts{
		NewBaseOpts: msp.NewBaseOpts{
			Version: msp.MSPv1_0,
		},
	}

	thisMSP, err := msp.New(mspOpts, cp)
	if err != nil {
		return nil, err
	}

	err = thisMSP.Setup(conf)
	if err != nil {
		return nil, err
	}

	return thisMSP, nil
}

// getPubKeyFromPemData looks for ECDSA public key in pemContent, and returns pem content only with the public key.
func getPubKeyFromPemData(pemContent []byte) ([]byte, error) {
	for {
		block, rest := pem.Decode(pemContent)
		if block == nil {
			break
		}
		pemContent = rest

		key, err := configtxgen.ParseCertificateOrPublicKey(block.Bytes)
		if err != nil {
			continue
		}

		return pem.EncodeToMemory(&pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: key,
		}), nil
	}

	return nil, errors.New("no ECDSA public key in pem file")
}

func createNamespacesTx(nsPolicy *applicationpb.NamespacePolicy, nsID string, nsVersion int) *applicationpb.Tx {
	writeToMetaNs := &applicationpb.TxNamespace{
		NsId: committerpb.MetaNamespaceID,
		// TODO we need the correct version of the metaNamespaceID
		NsVersion:  uint64(0),
		ReadWrites: make([]*applicationpb.ReadWrite, 0, 1),
	}

	policyBytes := protoutil.MarshalOrPanic(nsPolicy)
	rw := &applicationpb.ReadWrite{
		Key:   []byte(nsID),
		Value: policyBytes,
	}

	// note that we only set the version if we update a namespace policy
	if nsVersion >= 0 {
		rw.Version = applicationpb.NewVersion(uint64(nsVersion))
	}

	writeToMetaNs.ReadWrites = append(writeToMetaNs.ReadWrites, rw)

	tx := &applicationpb.Tx{
		Namespaces: []*applicationpb.TxNamespace{
			writeToMetaNs,
		},
	}

	return tx
}

func createSignedEnvelope(signer msp.SigningIdentity, channel string, tx *applicationpb.Tx) (*cb.Envelope, error) {
	signatureHdr := protoutil.NewSignatureHeaderOrPanic(signer)
	txID := protoutil.ComputeTxID(signatureHdr.Nonce, signatureHdr.Creator)

	tx, err := endorse(signer, txID, tx)
	if err != nil {
		return nil, err
	}

	channelHdr := protoutil.MakeChannelHeader(cb.HeaderType_MESSAGE, 0, channel, 0)
	channelHdr.TxId = txID

	payloadHdr := protoutil.MakePayloadHeader(channelHdr, signatureHdr)
	txBytes := protoutil.MarshalOrPanic(tx)
	return createEnvelope(signer, payloadHdr, txBytes)
}

func endorse(signer msp.SigningIdentity, txID string, tx *applicationpb.Tx) (*applicationpb.Tx, error) {
	if tx == nil {
		return nil, errors.New("nil transaction")
	}

	// check that tx does not yet carry any endorsements
	if tx.Endorsements == nil {
		tx.Endorsements = make([]*applicationpb.Endorsements, len(tx.GetNamespaces()))
	}

	// get signer signerCert
	signerID, err := getSignerID(signer)
	if err != nil {
		return nil, err
	}

	// create signature for each namespace in transaction
	for idx := range tx.GetNamespaces() {
		// Note that a default msp signer hash the msg before signing.
		// For that reason we use the TxNamespace message as ASN1 encoded msg

		msg, err := tx.Namespaces[idx].ASN1Marshal(txID)
		if err != nil {
			return nil, fmt.Errorf("failed asn1 marshal tx: %w", err)
		}

		sig, err := signer.Sign(msg)
		if err != nil {
			return nil, fmt.Errorf("failed signing tx: %w", err)
		}

		// store signature as endorsementWithIdentity
		eid := &applicationpb.EndorsementWithIdentity{
			Endorsement: sig,
			Identity:    signerID,
		}

		// check if there is already an endorsement for this namespace, so we can append the new endorsement
		// if not we create an empty endorser set
		if tx.Endorsements[idx] == nil {
			tx.Endorsements[idx] = &applicationpb.Endorsements{
				EndorsementsWithIdentity: []*applicationpb.EndorsementWithIdentity{},
			}
		}

		tx.Endorsements[idx].EndorsementsWithIdentity = append(tx.Endorsements[idx].EndorsementsWithIdentity, eid)
	}

	return tx, nil
}

func getSignerID(signer msp.SigningIdentity) (*msppb.Identity, error) {
	if signer == nil {
		return nil, errors.New("nil signer")
	}

	signerCert, err := signer.GetCertificatePEM()
	if err != nil {
		return nil, err
	}
	return msppb.NewIdentity(signer.GetIdentifier().Mspid, signerCert), nil
}

// createEnvelope creates a signed envelope from the passed header and data.
func createEnvelope(signer identity.SignerSerializer, hdr *cb.Header, data []byte) (*cb.Envelope, error) {
	payloadBytes := protoutil.MarshalOrPanic(
		&cb.Payload{
			Header: hdr,
			Data:   data,
		},
	)

	var sig []byte
	if signer != nil {
		var err error
		sig, err = signer.Sign(payloadBytes)
		if err != nil {
			return nil, err
		}
	}

	env := &cb.Envelope{
		Payload:   payloadBytes,
		Signature: sig,
	}

	return env, nil
}

func broadcast(odererCfg OrdererConfig, env *cb.Envelope) error {
	cl, err := comm.NewClient(odererCfg.Config)
	if err != nil {
		return fmt.Errorf("cannot get grpc client: %w", err)
	}

	conn, err := cl.NewDialer(odererCfg.OrderingEndpoint)()
	if err != nil {
		return fmt.Errorf("cannot get grpc client: %w", err)
	}
	defer func() {
		_ = conn.Close()
	}()

	occ := ab.NewAtomicBroadcastClient(conn)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	abc, err := occ.Broadcast(ctx)
	if err != nil {
		return err
	}

	err = abc.Send(env)
	if err != nil {
		return err
	}

	status, err := abc.Recv()
	if err != nil {
		return err
	}

	if status.GetStatus() != cb.Status_SUCCESS {
		return fmt.Errorf("got error %#v", status.GetStatus())
	}

	return nil
}
