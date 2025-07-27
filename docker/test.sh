#!/bin/sh
set -e

cleanup() {
  echo "Shutting down..."
  pkill -f "vault server" || true
  exit 0
}

trap cleanup INT TERM

MNEMONIC="die hard"

VAULT_ADDR="http://127.0.0.1:8200"
INIT_FILE="/vault/file/init-keys.json"
PLUGIN_NAME="vaultpoly"
PLUGIN_PATH="vaultpoly"
PLUGIN_SHA256=$(cat /vault/plugins/SHA256SUMS | awk '{print $1}')

# Start Vault server in the background if not running
if ! pgrep -f "vault server" >/dev/null 2>&1; then
  echo "Starting Vault server..."
  vault server -config=/vault/config/config.hcl &
fi

wait_for_vault() {
  until curl -s $VAULT_ADDR/v1/sys/health | grep -q '"initialized":'; do
    echo "Waiting for Vault to be ready..."
    sleep 2
  done
}

 

init_vault() {
  if ! vault status | grep -q "Initialized.*true"; then
    echo "Initializing Vault..."
    vault operator init -key-shares=3 -key-threshold=2 -format=json > "$INIT_FILE"
    echo "Vault initialized. Unseal keys and root token:"
    cat "$INIT_FILE"
  else
    echo "Vault already initialized."
    wait
    return
  fi
}

unseal_vault() {
  UNSEAL_KEYS=$(jq -r '.unseal_keys_b64[]' "$INIT_FILE")
  for key in $UNSEAL_KEYS; do
    vault operator unseal "$key" || true
  done
}

login_root() {
  export VAULT_TOKEN=$(jq -r '.root_token' "$INIT_FILE")
}
register_plugin() {
  if vault plugin list -output-curl-string | grep -q "$PLUGIN_NAME"; then
    return
  fi

  if ! vault plugin register -sha256="$PLUGIN_SHA256" secret "$PLUGIN_NAME"; then
    echo "Error: Failed to register plugin $PLUGIN_NAME" >&2
    exit 1
  fi
}

enable_plugin() {
  vault secrets list | grep -q "$PLUGIN_PATH/" && return
  vault secrets enable -plugin-name="$PLUGIN_NAME" -path="$PLUGIN_PATH" plugin
}


demo_eth(){
  # Create a wallet (Ethereum)
  echo "Creating Ethereum wallet..."
  CREATE_WALLET=$(vault write -format=json $PLUGIN_PATH/wallets/eth mnemonic='$MNEMONIC')
  ADDRESS=$(echo $CREATE_WALLET | jq -r '.data.address')
  echo "Created ETH wallet: $ADDRESS"

  # List wallets
  echo "Listing Ethereum wallets..."
  vault list $PLUGIN_PATH/wallets/eth || true

  # Sign a transaction (Ethereum)
  echo "Signing Ethereum transaction..."
  cat > payload.json <<EOF
  {
    "chainId": 97,
    "to": "0x0000000000000000000000000000000000000000",
    "value": 0,
    "data": "0x",
    "nonce": 0,
    "gas": 60000,
    "gasPrice": 1000000000
  }
EOF

  vault write $PLUGIN_PATH/wallets/eth/$ADDRESS/sign payload=@payload.json

  # Clean up
  rm -f payload.json
}


main() {
  wait_for_vault
  init_vault
  unseal_vault
  login_root
  register_plugin
  enable_plugin
  demo_eth
  echo "Vault setup complete."
}

main