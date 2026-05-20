<!-- SPDX-License-Identifier: Apache-2.0 -->
# Namespaces

## Overview

A namespace is the Fabric-X unit for logical state isolation and endorsement policy binding. Fabric-X uses a single channel model; namespaces separate application state inside that channel.

Canonical policy details live in [Namespace Policy](https://hyperledger.github.io/fabric-x-committer/namespace-policy/).

## What a Namespace Provides

- Separate key space for application state.
- Namespace-specific endorsement policy.
- Policy verification during commit through the Verifier service.
- MVCC validation and writes through namespace tables managed by the Validator-Committer.

## Namespace Policies

Every namespace that can be modified must have a namespace policy. Fabric-X supports two policy forms:

| Policy type | Purpose |
| --- | --- |
| Threshold rule | Lightweight single-signer policy using raw public key and signature scheme. |
| MSP rule | Fabric-compatible `SignaturePolicyEnvelope` over MSP principals and combinators. |

Policies are stored and propagated through the committer path. The Coordinator sends policy updates to Verifier services after relevant committed changes.

## Single Channel Model

Fabric-X does not use Fabric-style multi-channel isolation. Use namespaces to separate application domains while keeping one ordered transaction stream.

## Boundaries

Namespaces are not independent ledgers. They share ordering, committer services, and database infrastructure. Cross-namespace transactions can exist when transaction read/write sets touch multiple namespaces, and validation handles each namespace according to its policy and state versions.

## Operational Note

This concept page does not define a namespace-management CLI. Use source repository tooling and configuration docs for current operational procedures.

## See Also

- [Namespace Policy](https://hyperledger.github.io/fabric-x-committer/namespace-policy/)
- [Policies](policies/policies.md)
- [Validator-Committer](../committer/docs/validator-committer.md)
- [Committer Pipeline](../committer/docs/architecture.md)
