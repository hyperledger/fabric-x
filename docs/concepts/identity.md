<!-- SPDX-License-Identifier: Apache-2.0 -->
# Identity Management

## Overview

Fabric-X is a permissioned system, so identities are first-class network objects. Every client, service, administrator, and endorsing participant acts through a cryptographic identity. That identity is used to sign messages, authenticate transport sessions, satisfy policies, and create audit trails.

Fabric-X inherits Fabric-compatible identity concepts through `fabric-x-common`. The primary identity format is an X.509 certificate issued by a trusted Certificate Authority (CA) and validated through a Membership Service Provider (MSP).

An identity proves control of private key material. It does not, by itself, authorize an action. Authorization comes from policy evaluation after the identity has been validated.

## Actors and Identities

Fabric-X identities are used by many actors:

| Actor | Identity use |
| --- | --- |
| Clients | Submit transactions, query state, and receive status. |
| FSC nodes | Run application views, coordinate protocols, and sign transaction material. |
| Custom endorsers | Execute application-specific endorsement logic and sign read/write sets. |
| Arma services | Authenticate ordering-service traffic and sign ordering messages where configured. |
| Committer services | Authenticate validation, commit, sidecar, and query-service communication. |
| Query services | Authenticate callers and service-to-service traffic. |
| Administrators | Sign configuration updates, MSP changes, and policy changes. |

A valid certificate says who produced a signature according to MSP trust rules. A policy says whether that identity is allowed to perform the requested action.

## X.509 Certificates

An X.509 certificate binds a public key to subject information and an issuer. The certificate chain links that identity to a root or intermediate CA trusted by an MSP.

Common certificate uses in Fabric-X include:

- transaction submission signatures;
- endorsement signatures;
- configuration update signatures;
- service enrollment identities;
- TLS authentication identities;
- MSP principal evaluation;
- namespace-policy verification.

Production deployments should separate concerns where appropriate. TLS certificates and enrollment/signing certificates can be issued and rotated independently so transport security changes do not necessarily change application signing identities.

## Certificate Authorities

Certificate Authorities issue certificates for organizations, users, and services. Fabric-X deployments can use Fabric CA or an enterprise PKI as long as issued certificates are compatible with MSP configuration.

Certificates must chain to trusted roots or intermediates and carry the organizational information and organizational units expected by policy. Certificate lifecycle management is an operational responsibility: organizations need enrollment, renewal, revocation, emergency replacement, and audit procedures.

## Signing Identities

A signing identity combines a certificate with private key material. Clients, FSC nodes, custom endorsers, Arma services, committer services, query services, and administrators use signing identities to sign messages they produce.

Private keys must be protected with filesystem permissions, HSMs, cloud KMS integrations, or equivalent production controls. A signing identity should be scoped to its job. Do not reuse an administrative identity for routine transaction submission, endorsement, or service-to-service traffic.

## TLS Identities

TLS identities authenticate network connections. Production deployments should use mutual TLS for node-to-node and service-to-service communication so both sides prove identity before exchanging traffic.

TLS trust and signing trust are related but not identical. A certificate that is valid for TLS does not automatically satisfy endorsement, namespace, or configuration policy.

## Roles and Node OUs

MSPs can classify identities into roles using Node Organizational Units (Node OUs). Common roles include:

| Role | Meaning |
| --- | --- |
| client | Application or user identity that submits transactions or queries. |
| admin | Administrative identity allowed to approve governance operations. |
| peer | Fabric-compatible role that may be referenced by existing policy structures. |
| orderer | Ordering-service identity role. |
| member | General organization member identity. |

Policies use these roles to express authorization. A namespace policy can require signatures from organization members, while a configuration update can require organization admins.

## Identity Validation

Identity validation starts with MSP trust rules. The MSP checks certificate chains, trusted roots and intermediates, role mappings, expiration, and revocation data. If the identity is valid, policy evaluation can decide whether signatures from that identity satisfy required principals.

Fabric-X namespace-policy verification can also use cached identity references to avoid carrying full certificates in every endorsement. The verifier resolves cached references to identities and then performs normal signature and policy checks.

## Revocation and Expiry

MSP validation can reject identities based on certificate expiry, revocation lists, or trust-chain changes. Operators should keep CA roots, intermediates, certificate revocation lists, and Node OU configuration synchronized across participants.

Revocation matters for long-running networks. If a key is compromised or an organization changes personnel, the network must be able to stop accepting signatures from affected certificates according to configured MSP rules.

## See Also

- [Membership Service Provider](msp.md)
- [Endorsement and Other Policies](policies/policies.md)
- [Verifier](../committer/docs/verification-service.md)
- `fabric-x-common/msp`
