<!--
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
-->
# Fabric-X Common

This library encapsulates some common code across Fabric-X ecosystem.
This code have been taken from [Hyperledger Fabric](https://github.com/hyperledger/fabric) v3.0.0-rc1.

### Config
Uses [Hyperledger Fabric Config](https://github.com/hyperledger/fabric-config) v0.3.0

- common/configtx
- internaltools/configtxgen

This modifies Fabric's config block:
* Added ARMA field for orderer type
* Added `MetaNamespacePolicyKey` field

### MSP
- hyperledger/fabric/msp
- hyperledger/fabric/common/channelconfig
    - bundle.go: (b *Bundle) ValidateNew(nb Resources)
- hyperledger/fabric/common/capabilities
- hyperledger/fabric/common/configtx
- hyperledger/fabric/common/policies

### Tools
- hyperledger/fabric/cmd/configtxgen
- hyperledger/fabric/cmd/configtxlator
- hyperledger/fabric/cmd/osnadmin
