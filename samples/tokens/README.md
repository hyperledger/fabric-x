<!--
SPDX-License-Identifier: Apache-2.0
-->
# Token SDK Sample

The **Token SDK Sample** demonstrates how to:

- Build a simple token-based application using the [Token SDK](https://github.com/hyperledger-labs/fabric-token-sdk).
- Connect the application to both [Fabric-X](https://github.com/hyperledger/fabric-x) and classic [Fabric](https://github.com/hyperledger/fabric) networks.
- Issue and transfer tokens via a REST API.

## About the Sample

This demo provides a set of services exposing REST APIs that integrate with the [Token SDK](https://github.com/hyperledger-labs/fabric-token-sdk)
to issue, transfer, and redeem tokens backed by a **Hyperledger Fabric(x)** network for validation and settlement.

Together, these services form a *Layer 2 network* capable of transacting privately among participants.
The ledger data does not reveal balances, transaction amounts, or participant identities.
Tokens are represented as UTXOs owned by pseudonymous keys, with details hidden through **Zero-Knowledge Proofs (ZKPs)**.

The application follows the Fabric-X programming model, where business parties directly endorse transactions—rather than Fabric peers executing chaincode.
Note that the [Token SDK](https://github.com/hyperledger-labs/fabric-token-sdk) builds on top of the [Fabric Smart Client](https://github.com/hyperledger-labs/fabric-smart-client) (FSC), a framework to build distributed applications for Fabric(x).

This sample helps you get familiar with Token SDK features and serves as a starting point for your own proof of concept.

**Components**

**Application services**
- **Issuer service** - creates (issues) tokens.
- **Owner services** - host user wallets.
- **Endorser service** - validates and approves token transactions.


**Fabric(x) Blockchain Network**
- An offline Certificate Authority (CA).
- Configuration for a **Fabric-X** test network.
- Configuration for a **Fabric v3** test network.

Below is a high level overview of the components and how data flows in a token transfer transaction.
The sequence diagram later in this readme provides more details about the token transaction.
For a specification of the Fabric-X components and their interactions, refer to the main [README.md](../../README.md).

![transfer: high level](./diagrams/components.png)

## Architecture Overview

From now on, we’ll refer to the issuer, endorser, and owner services collectively as nodes (not to be confused with Fabric peer nodes).

Each node runs as a separate application with:

- A REST API
- The FSC node runtime
- The Token SDK

Nodes communicate via *websockets* to construct token transactions.
Each node also acts as a Fabric user, submitting transactions to the settlement layer — any Fabric or Fabric-X network.

A namespace (`token_namespace`) is deployed, along with a committed transaction containing the identities of the issuer, endorsers, and CA, enabling transaction validation.

## Prerequisites

### Fabric-X Setup

We use the [Fabric-x Ansible Collection](https://github.com/LF-Decentralized-Trust-labs/fabric-x-ansible-collection?tab=readme-ov-file#option-2-install-from-source) to set up the Fabric-X test network.
Please check the installation guidelines for more details.

#### Requirements

- `python`;
- [`ansible`](https://docs.ansible.com/ansible/latest/installation_guide/intro_installation.html) >= **2.16**;
- [`podman`](https://podman.io/docs/installation) or [`docker`](https://docs.docker.com/engine/install/);
- [`go`](https://go.dev/doc/install).

#### Installation

Clone the repository (anywhere on your machine) and install the ansible collection.

```bash
git clone https://github.com/LF-Decentralized-Trust-labs/fabric-x-ansible-collection.git
cd fabric-x-ansible-collection
make install
```

Back in the Token SDK Sample directory:

```shell
make install-prerequisites
python3 -m pip install -r ansible/requirements.txt
```

> [!NOTE]
> Mac users:, Fabric-X components must communicate via host.docker.internal instead of localhost.
> ```shell
> export LOCAL_ANSIBLE_HOST="host.docker.internal"
> ```

We use the Fabric CA for issuing idemix credentials for the accounts. Download it [here](https://github.com/hyperledger/fabric-ca/releases)
or follow the v3 instructions below to install it (and make sure that it is in your `PATH`).

### Fabric v3 Setup

To run on Fabric v3, we use Fabric samples test network (`$(FABRIC_SAMPLES)/test-network/network.sh`).

Ensure Fabric binaries are in your `PATH` and Docker images are available. If not, install them as follows. 

```shell
./install-fabric.sh --fabric-version 3.1.1
export PATH=$(pwd)/fabric-samples/bin:$PATH
```

> [!TIP]
> The scripts assume you have run the script from the directory of this README. If you have them somewhere else,
> just make sure to `export FABRIC_SAMPLES="/path/to/fabric-samples"` so the scripts can start the test network.

## Getting Started

> [!NOTE]
> The sample uses Fabric-X as the default network.
> If you want to run the application on Fabric v3, just set the PLATFORM to fabric3.
>
> ```bash
> export PLATFORM=fabric3
> ```
> 
> When switching between platforms, always make sure to *first* (before changing the platform) clean up all artifacts:
> ```shell
> make teardown
> make clean
> ```

### Generate Crypto Material

Create the configurations and crypto material for the network:

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

The relevant crypto material is copied to the folders in the conf/* directories.

### Start the Network and Application

Start the Fabric network, create the namespace, and start the application services.

```shell
make start-fabric
make create-namespace
make start-app
```

Or simply:

```shell
make start
```

### Interacting with the Application

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

#### Example: Issue tokens

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

#### Example: Transfer tokens

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

#### UTXO Model

Note that the application uses the UTXO model (like bitcoin).
- The issuer creates a token of `1000 TOK` owned by `alice`.
- When `alice` transfers `100 TOK` to `dan`, her `1000 TOK` token becomes the **input**.
- Two **outputs** are created:
  1) `100 TOK` owned by `dan`
  2) `900 TOK` owned by `alice`

Every transfer consumes existing outputs and creates new ones, ensuring balance consistency.

### Deep Dive: What Happens During a Transfer?

Let’s examine how a private token transfer works between `alice` (Owner 1) and `dan` (Owner 2):

1. **Create Transaction:**

    Alice requests an anonymous key from Dan, creates commitments that can be verified by anyone, but _only_ be opened (read) by Dan.
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

### Teardown and cleanup

Convenient Make targets are provided for shutting down, restarting, and cleaning the environment.

Run:

```bash
make help
```

for a list of available commands.

## Debug mode

For a faster development, you can run the services outside Docker.

First, add the following to `/etc/hosts`:

```
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

The application services discovers the peer addresses from the channel configuration after connecting to committer-queryservice (or a trusted peer in Fabric v3).

Next, start the network as before:

```bash
make start-fabric
make create-namespace
# don't make start-app
```

In separate terminals:

```bash
cd conf/issuer && go run ../../issuer --port 9100
cd conf/endorser1 && go run ../../endorser --port 9300
cd conf/owner && go run ../../owner --port 9500
```

### VSCode

If you use VSCode, copy:

```bash
mkdir -p ../../.vscode
cp launch.example.json ../../.vscode/launch.json
```

Then run or debug the application services directly.
