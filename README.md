# VaultPoly

VaultPoly is a HashiCorp Vault plugin for managing blockchain wallets and signing transactions for Ethereum and Bitcoin. It provides secure wallet creation, storage, and transaction signing via Vault API endpoints.

## Features

- Create and manage Ethereum and Bitcoin wallets
- Sign Ethereum and Bitcoin transactions
- Secure storage of private keys using Vault
- Extensible adapter-based architecture

## Requirements

- Go 1.20+
- HashiCorp Vault 1.8+

## Setup

### 1. Build the Plugin

```
git clone https://github.com/igwedaniel/vaultpoly.git
cd vaultpoly
make
```

This will build the plugin binary.

### 2. Register the Plugin with Vault

Copy the built binary to your Vault plugins directory and register it:

```
export VAULT_ADDR='http://127.0.0.1:8200'
vault plugin register -sha256="$(sha256sum vaultpoly | awk '{print $1}')" \
  secret vaultpoly
```

### 3. Enable the Plugin

```
vault secrets enable -path=vault-poly \
  -plugin-name=vaultpoly plugin
```

## API Usage

### Create a Wallet

**Endpoint:** `POST /v1/vault-poly/wallets/<blockchainType>`

- `blockchainType`: `eth` or `btc`

**Example:**

```
curl --header "X-Vault-Token: <token>" \
     --request POST \
     --data '{"blockchainType": "eth"}' \
     http://127.0.0.1:8200/v1/vault-poly/wallets/eth
```

**Response:**

```
{
  "data": {
    "address": "<wallet_address>"
  }
}
```

### List Wallets

**Endpoint:** `LIST /v1/vault-poly/wallets/<blockchainType>`

**Example:**

```
curl --header "X-Vault-Token: <token>" \
     --request LIST \
     http://127.0.0.1:8200/v1/vault-poly/wallets/eth
```

### Sign a Transaction

**Endpoint:** `POST /v1/vault-poly/wallets/<blockchainType>/<address>/sign`

- `blockchainType`: `eth` or `btc`
- `address`: Wallet address
- `payload`: JSON-encoded transaction payload (see below)

**Example (Ethereum):**

```
curl --header "X-Vault-Token: <token>" \
     --request POST \
     --data '{"payload": "{...}"}' \
     http://127.0.0.1:8200/v1/vault-poly/wallets/eth/<address>/sign
```

**Example (Bitcoin):**

```
curl --header "X-Vault-Token: <token>" \
     --request POST \
     --data '{"payload": "{...}"}' \
     http://127.0.0.1:8200/v1/vault-poly/wallets/btc/<address>/sign
```

#### Ethereum Payload Example

```
{
  "chainId": 97,
  "to": "0x...",
  "value": 0,
  "data": "0x...",
  "nonce": 0,
  "gas": 60000,
  "gasPrice": 1000000000
}
```

#### Bitcoin Payload Example

```
{
  "recipient": "tb1q...",
  "amount": 100000,
  "fee_rate": 1.0,
  "utxos": [
    {
      "txid": "...",
      "vout": 1,
      "value": 500000,
      "script_pub_key": "...",
      "script_pubkey_type": "v0_p2wpkh"
    }
  ]
}
```

## Testing

Run all tests:

```
make test
```

Run a specific test:

```
make test-case name="TestWalletSign/Sign_Wallet_ETH_-_pass"
```

## Extending

Adapters for new blockchains can be added by implementing the `BlockchainAdapter` interface in `internal/adapters/adapters.go` and updating the adapter selector.

## License

MIT
