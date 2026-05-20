<!-- SPDX-License-Identifier: Apache-2.0 -->
# Membership Service Provider (MSP)

## Overview

The Membership Service Provider (MSP) defines how identities are represented, validated, and mapped to policy principals. Fabric-X uses Fabric-compatible MSP validation from `fabric-x-common/msp` and Fabric-compatible policy structures where MSP rules are used.

An MSP is the bridge between certificates and governance. It tells the network which certificate authorities are trusted for an organization, which certificates are administrators, how roles are classified, and how signatures are verified during policy evaluation.

## What an MSP Does

An MSP is responsible for:

- loading trusted root and intermediate CA certificates;
- validating certificate chains;
- checking certificate revocation data;
- mapping identities to roles such as member, admin, client, peer, or orderer;
- deserializing identities from transaction or configuration data;
- verifying signatures against identity public keys;
- providing MSP principals for policy evaluation.

MSP validation answers: "Does this certificate belong to this organization according to configured trust rules?" Policy evaluation then answers: "Is this trusted identity allowed to perform this action?"

## MSP Identifier

Each MSP has an MSP ID. Policies use this identifier when referring to organizational principals. MSP IDs must be unique within network configuration so signatures can be attributed unambiguously.

For example, a policy might refer to `Org1MSP.member`, `Org2MSP.admin`, `BankMSP.client`, or `OrdererMSP.orderer`. The MSP ID selects the organization trust domain; the role selects which class of identity inside that trust domain is acceptable.

## MSP Configuration

A typical MSP configuration includes:

| Item | Purpose |
| --- | --- |
| Root CA certificates | Trust anchors for identity validation. |
| Intermediate CA certificates | Delegated issuers trusted under roots. |
| Admin certificates or admin OU rules | Identities allowed to administer the organization. |
| Node OU configuration | Mapping from certificate OUs to roles. |
| CRLs | Revoked certificates that must no longer be trusted. |
| TLS CA certificates | Trust roots for TLS identities where configured. |

The exact folder layout can vary by tooling, but the purpose remains the same: define the organization's trust boundary.

## Organizational MSP and Local MSP

| MSP use | Purpose |
| --- | --- |
| Organizational MSP | Defines public trust and policy principals for an organization. Other parties use it to validate signatures from that organization. |
| Local MSP | Holds private signing material for one node, service, admin, or client process. The local process uses it to sign messages. |

Organizational MSP material is shared through network configuration. Local MSP private keys are never shared.

## Node OUs and Roles

Node OU configuration lets an MSP classify certificates by role. Instead of listing every admin or service certificate explicitly, an organization can define OU-based rules such as "certificates issued by this CA with OU=admin are admins."

Common classifications:

- **client**: submits transactions or queries;
- **admin**: approves administrative operations;
- **peer**: satisfies peer-style principals used by Fabric-compatible policies;
- **orderer**: identifies ordering-service nodes;
- **member**: any valid identity under the MSP.

Node OUs make policies easier to maintain because adding a new certificate does not require changing every policy. The certificate must still be issued by a trusted CA and match configured OU rules.

## MSP Principals

Policies do not usually name raw certificates. They name MSP principals. A principal is a condition an identity must satisfy, such as membership in an MSP or possession of an admin role.

Examples:

- a member of `Org1MSP`;
- an admin of `Org2MSP`;
- a client identity from `BankMSP`;
- an orderer identity from `OrdererMSP`;
- a peer-role identity from an application operator MSP.

During policy evaluation, signatures are matched to identities, identities are validated by MSPs, and those identities are checked against principals in the policy tree.

## MSPs in Fabric-X Policies

Namespace MSP rules use Fabric `SignaturePolicyEnvelope` structures. During verification:

1. transaction endorsements carry identities or cached identity IDs;
2. the committer verifier deserializes identities through MSP logic;
3. signatures are checked against identity public keys;
4. identities are matched to MSP principals;
5. the policy tree is evaluated against the collected signatures.

This lets namespace policy express multi-organization governance such as "Org1 and Org2 must endorse" or "any two of Org1, Org2, and Org3 must endorse." Configuration policies use the same identity foundation for MSP updates, organization changes, ordering settings, and namespace metadata changes.

## Best Practices

- Use one MSP per organization unless there is a clear governance reason to split trust domains.
- Keep MSP IDs stable; changing them affects policy references.
- Prefer Node OU role mapping for maintainable role assignment.
- Keep root CAs, intermediate CAs, admin definitions, and CRLs synchronized across participants.
- Protect local MSP private keys with production key management.
- Treat MSP updates as high-impact governance changes.

## See Also

- [Identity Management](identity.md)
- [Endorsement and Other Policies](policies/policies.md)
- [Namespace Policy](https://hyperledger.github.io/fabric-x-committer/namespace-policy/)
- `fabric-x-common/msp`
- `fabric-x-common/common/policies`
