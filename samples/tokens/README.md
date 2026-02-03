<!--
SPDX-License-Identifier: Apache-2.0
-->

# Token SDK Sample

The **Token SDK Sample** demonstrates how to:

- Build a simple token-based application using the [Token SDK](https://github.com/hyperledger-labs/fabric-token-sdk).
- Connect the application to both [Fabric-X](https://github.com/hyperledger/fabric-x) and classic [Fabric](https://github.com/hyperledger/fabric) networks.
- Issue and transfer tokens via a REST API.

## Table of Contents

- [Table of Contents](#table-of-contents)
- [About the Sample](#about-the-sample)
  - [Components](#components)
    - [Application services](#application-services)
    - [Fabric(x) Blockchain Network](#fabricx-blockchain-network)
  - [Application](#application)
  - [UTXO Model](#utxo-model)
  - [Deep Dive: What Happens During a Transfer?](#deep-dive-what-happens-during-a-transfer)
- [Running the sample](#running-the-sample)
- [Prerequisites](#prerequisites)
- [Default option: Fabric-X with Ansible](#default-option-fabric-x-with-ansible)
  - [Requirements](#requirements)
  - [Installation](#installation)
  - [Setup Fabric-X](#setup-fabric-x)
- [Option 2: Fabric-X test container](#option-2-fabric-x-test-container)
- [Option 3: Fabric v3](#option-3-fabric-v3)
  - [Setup Fabric v3](#setup-fabric-v3)
  - [Start the Network and Application](#start-the-network-and-application)
- [Interacting with the Application](#interacting-with-the-application)
- [Example: Issue tokens](#example-issue-tokens)
- [Example: Transfer tokens](#example-transfer-tokens)
- [Teardown and cleanup](#teardown-and-cleanup)
- [Development](#development)
- [Debug mode](#debug-mode)
  - [VSCode](#vscode)
  - [Running the binaries](#running-the-binaries)
- [Troubleshooting](#troubleshooting)

## About the Sample

This demo provides a set of services exposing REST APIs that integrate with the [Token SDK](https://github.com/hyperledger-labs/fabric-token-sdk)
to issue, transfer, and redeem tokens backed by a **Hyperledger Fabric(x)** network for validation and settlement.

Together, these services form a _Layer 2 network_ capable of transacting privately among participants.
The ledger data does not reveal balances, transaction amounts, or participant identities.
Tokens are represented as UTXOs owned by pseudonymous keys, with details hidden through **Zero-Knowledge Proofs (ZKPs)**.

The application follows the Fabric-X programming model, where business parties directly endorse transactions—rather than Fabric peers executing chaincode.
Note that the [Token SDK](https://github.com/hyperledger-labs/fabric-token-sdk) builds on top of the [Fabric Smart Client (FSC)](https://github.com/hyperledger-labs/fabric-smart-client), a framework to build distributed applications for Fabric(x).

This sample helps you get familiar with Token SDK features and serves as a starting point for your own proof of concept.

### Components

#### Application services

- **Issuer service** - creates (issues) tokens.
- **Owner services** - host user wallets.
- **Endorser service** - validates and approves token transactions.

#### Fabric(x) Blockchain Network

- An offline Certificate Authority (CA).
- Configuration for a **Fabric-X** test network.
- Configuration for a **Fabric v3** test network.

Below is a high level overview of the components and how data flows in a token transfer transaction.
The sequence diagram later in this readme provides more details about the token transaction.
For a specification of the Fabric-X components and their interactions, refer to the main [README.md](../../README.md).

![transfer: high level](./diagrams/components.png)

### Application

From now on, we’ll refer to the issuer, endorser, and owner services collectively as nodes (not to be confused with Fabric peer nodes).

Each node runs as a separate application with:

- A REST API
- The FSC node runtime
- The Token SDK

Nodes communicate via _websockets_ to construct token transactions.
Each node also acts as a Fabric user, submitting transactions to the settlement layer — any Fabric or Fabric-X network.

A namespace (`token_namespace`) is deployed, along with a committed transaction containing the identities of the issuer, endorsers, and CA, enabling transaction validation.

### UTXO Model

Note that the application uses the UTXO model (like bitcoin).

- The issuer creates a token of `1000 TOK` owned by `alice`.
- When `alice` transfers `100 TOK` to `dan`, her `1000 TOK` token becomes the **input**.
- Two **outputs** are created:
  1. `100 TOK` owned by `dan`
  2. `900 TOK` owned by `alice`

Every transfer consumes existing outputs and creates new ones, ensuring balance consistency.

The Token SDK exposes all transactions, including "change" (the remainder returned to the sender).

### Deep Dive: What Happens During a Transfer?

Let’s examine how a private token transfer works between `alice` (Owner 1) and `dan` (Owner 2):

1. **Create Transaction:**

   Alice requests an anonymous key from Dan, creates commitments that can be verified by anyone, but can _only_ be opened (read) by Dan.
   The commitments contain the value, sender and recipient of each of the in- and output tokens.

2. **Get Endorsements:**

   Alice submits the transaction to the endorser which validates the transaction using the token validation logic.
   In detail, it verifies that all the proofs are valid and all the necessary signatures are there.
   Note that the endorser cannot see the actual transfer details thanks to the zero knowledge proofs.

3. **Commit Transaction:**

   Alice submits the endorsed fabric(x) transaction to the ordering service.
   Once committed, all involved nodes (Owner 1, Owner 2) receive events and update the transaction status to `Confirmed.`
   The transaction is now final; Dan now officially owns the `100 TOK`.

![transfer](diagrams/transfer_transaction.png)

## Running the sample

## Prerequisites

You will need docker or podman to run the fabric network and application.

With the following command, we will download the Fabric 3 binaries, docker images and samples. For fabric-x based networks, we only use the Fabric CA for issuing idemix credentials for the accounts. If you want to run the same application against Fabric 3, this will provide you with the necessary prerequisites for that too.

```shell
make install-prerequisites
```

Make sure the CA binaries are accessible in your $PATH (add it to your .bashrc or .zshrc or equivalent for ease of use):

```shell
export PATH="$PATH:$(pwd)/fabric-samples/bin"
```

## Default option: Fabric-X with Ansible

Use ansible scripts to deploy real distributed networks. For the sake of this sample, we included a simple network that runs on your laptop, but there is a wealth of options to deploy to separate VMs with ease. Checkout the [Fabric-x Ansible Collection](https://github.com/LF-Decentralized-Trust-labs/fabric-x-ansible-collection?tab=readme-ov-file#option-2-install-from-source) to learn more.

### Requirements

- `python`;
- [`ansible`](https://docs.ansible.com/ansible/latest/installation_guide/intro_installation.html) >= **2.16**;

### Installation

```shell
make install-prerequisites
python3 -m pip install -r ansible/requirements.txt
```

### Setup Fabric-X

Make sure that the crypto is cleared and let the scripts know you want to use ansible:

```shell
make teardown
make clean
export PLATFORM=fabricx
make setup
```

Then, like with the test container:

```shell
make start
curl -X POST http://localhost:9300/endorser/init
```

## Option 2: Fabric-X test container

The quickest way to get going: a test version of Fabric-X in a single docker container! Even if you want to use different backends, we suggest to start here.

First generate the necessary crypto material:

```shell
make setup
```

This creates:

- Fabric
  - config files and identities for the orderers and committers
  - a genesis block
  - users that can submit or query transactions
  - endorser identity
- Fabric Smart Client
  - identities for the nodes (issuer, owner1, owner2, endorser)
- Fabric Token SDK
  - an idemix issuer for the token accounts
  - idemix credentials signed by this issuer
  - cryptographic parameters and configuration for the token network (see: `go tool tokengen pp print -i conf/namespace/zkatdlognoghv1_pp.json`).

The relevant crypto material is copied to the folders in the conf/\* directories.

The following first command starts the Fabric-X test container, creates a namespace, and runs the application in docker containers. The second command ensures that the parameters for the network (cryptographic material, the idemix issuer identity for the accounts, the token issuer certificate) are registered on the ledger.

```shell
make start
curl -X POST http://localhost:9300/endorser/init
```

Now open <http://localhost:8080> in your browser to see the other API endpoints, or scroll down to follow some `curl` commands.

## Option 3: Fabric v3

Run the same application against a classic Fabric v3 network.

### Setup Fabric v3

First, clean up any previous state and set Fabric v3:

```shell
make teardown
make clean
export PLATFORM=fabric3
make setup
```

### Start the Network and Application

Start the Fabric network, create the namespace (chaincode), and start the application services. For Fabric 3, you don't have to call the Init endpoint; this is taken care of when installing the chaincode.

```shell
make start
```

## Interacting with the Application

All services run as Docker containers and expose REST APIs.
They also communicate over P2P websockets as shown below:

| Rest API | P2P  | Service                     |
| -------- | ---- | --------------------------- |
| 8080     |      | API documentation (web)     |
| 9100     | 9101 | Issuer                      |
| 9300     | 9301 | Endorser 1                  |
| 9400     | 9401 | Endorser 2 (Fabric v3 only) |
| 9500     | 9501 | Owner 1 (alice and bob)     |
| 9600     | 9601 | Owner 2 (carlos and dan)    |

We can use the Swagger API on [http://localhost:8080](http://localhost:8080) or call the API directly via `curl`.

Now let's issue and transfer some tokens!

## Example: Issue tokens

We begin with initializing the token namespace (commit the parameters for the network) and issue `TOK` tokens to `alice`.

```bash
curl -X POST http://localhost:9300/endorser/init  # Fabric-X only

curl http://localhost:9100/issuer/issue --json '{
    "amount": {"code": "TOK","value": 1000},
    "counterparty": {"node": "owner1","account": "alice"},
    "message": "hello world!"
}'

curl http://localhost:9500/owner/accounts/alice | jq
curl http://localhost:9600/owner/accounts/dan | jq
```

## Example: Transfer tokens

Now `alice` transfers `100 TOK` to `dan`.

```bash
curl http://localhost:9500/owner/accounts/alice/transfer --json '{
    "amount": {"code": "TOK","value": 100},
    "counterparty": {"node": "owner2","account": "dan"},
    "message": "hello dan!"
}'

curl -X GET http://localhost:9600/owner/accounts/dan/transactions | jq
curl -X GET http://localhost:9500/owner/accounts/alice/transactions | jq
```

## Teardown and cleanup

To fully stop and delete the state of the application, run:

```shell
make teardown
```

To also delete the crypto (you'll have to run `make setup` again):

```shell
make clean
```

Convenient Make targets are provided for shutting down, restarting, and cleaning the environment.

Run:

```shell
make help
```

for a list of available commands.

## Development

## Debug mode

For faster development, you can run the services outside Docker.

First, add the following to `/etc/hosts`:

```text
127.0.0.1 peer0.org1.example.com
127.0.0.1 peer0.org2.example.com
127.0.0.1 orderer.example.com
127.0.0.1 issuer.example.com
127.0.0.1 endorser1.example.com
127.0.0.1 endorser2.example.com
127.0.0.1 owner1.example.com
127.0.0.1 owner2.example.com
127.0.0.1 committer-sidecar
127.0.0.1 committer-queryservice
127.0.0.1 host.docker.internal
```

The application services discover the peer addresses from the channel configuration after connecting to committer-queryservice (or a trusted peer in Fabric v3).

Next, start the network as before, but instead of `make start`, do:

```bash
make start-fabric
# don't make start-app
```

### VSCode

If you use VSCode, copy:

```bash
mkdir -p ../../.vscode
cp launch.example.json ../../.vscode/launch.json
```

Then run or debug the application services directly.

### Running the binaries

In separate terminals:

```bash
cd conf/issuer && go run ../../issuer --port 9100
cd conf/endorser1 && go run ../../endorser --port 9300
cd conf/owner && go run ../../owner --port 9500
```

## Troubleshooting

If the application doesn't work, there's a good chance that it has to do with stale keys or data. Your best bet is to:

```shell
make teardown
make clean
make setup
```

Before running `make start` again.

Otherwise, take a look at the logs. Note that an error down the line could be caused by an issue at startup, often a misconfiguration.
