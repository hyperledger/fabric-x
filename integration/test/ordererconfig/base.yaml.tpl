#
# Copyright IBM Corp. All Rights Reserved.
#
# SPDX-License-Identifier: Apache-2.0
#

PartyID: PARTY_ID
General:
  ListenAddress: 0.0.0.0
  ListenPort: PORT
  TLS:
    Enabled: true
    PrivateKey: ARTIFACTS_DIR/ordererOrganizations/ORG_DOMAIN/orderers/PARTY/NODE_DIR/tls/server.key
    Certificate: ARTIFACTS_DIR/ordererOrganizations/ORG_DOMAIN/orderers/PARTY/NODE_DIR/tls/server.crt
    RootCAs:
      - ARTIFACTS_DIR/ordererOrganizations/ORG_DOMAIN/msp/tlscacerts/tlsca.ORG_DOMAIN-cert.pem
    ClientAuthRequired: true
    ClientRootCAs:
      - ARTIFACTS_DIR/ordererOrganizations/orderer-org-1/msp/tlscacerts/tlsca.orderer-org-1-cert.pem
      - ARTIFACTS_DIR/ordererOrganizations/orderer-org-2/msp/tlscacerts/tlsca.orderer-org-2-cert.pem
      - ARTIFACTS_DIR/ordererOrganizations/orderer-org-3/msp/tlscacerts/tlsca.orderer-org-3-cert.pem
      - ARTIFACTS_DIR/ordererOrganizations/orderer-org-4/msp/tlscacerts/tlsca.orderer-org-4-cert.pem
      CLIENT_ROOT_CAS_EXTRA
  Keepalive:
    ClientInterval: 1m0s
    ClientTimeout: 20s
    ServerInterval: 2h0m0s
    ServerTimeout: 20s
    ServerMinInterval: 1m0s
  Backoff:
    BaseDelay: 1s
    Multiplier: 1.6
    MaxDelay: 2m0s
  MaxRecvMsgSize: 104857600
  MaxSendMsgSize: 104857600
  Bootstrap:
    Method: block
    File: ARTIFACTS_DIR/config-block.pb.bin
  Cluster:
    SendBufferSize: 2000
    ClientCertificate: ARTIFACTS_DIR/ordererOrganizations/ORG_DOMAIN/orderers/PARTY/NODE_DIR/tls/server.crt
    ClientPrivateKey: ARTIFACTS_DIR/ordererOrganizations/ORG_DOMAIN/orderers/PARTY/NODE_DIR/tls/server.key
  LocalMSPDir: ARTIFACTS_DIR/ordererOrganizations/ORG_DOMAIN/orderers/PARTY/NODE_DIR/msp
  LocalMSPID: ORG_MSP_ID
  LogSpec: info
FileStore:
  Location: STORAGE_DIR
