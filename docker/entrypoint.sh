#!/bin/sh

set -e

VAULT_ADDR="http://127.0.0.1:8200"
INIT_FILE="/vault/file/init-keys.json"
PLUGIN_NAME="vaultpoly"
PLUGIN_PATH="vaultpoly"
PLUGIN_SHA256=$(cat /vault/plugins/SHA256SUMS | awk '{print $1}')
VAULT_PID=""

cleanup() {
  if [ -n "$VAULT_PID" ]; then
    kill "$VAULT_PID" >/dev/null 2>&1 || true
    wait "$VAULT_PID" 2>/dev/null || true
  fi
  exit 0
}

trap cleanup INT TERM

start_vault() {
  echo "Starting Vault server..."
  vault server -config=/vault/config/config.hcl &
  VAULT_PID=$!
}

stop_vault() {
  if [ -n "$VAULT_PID" ]; then
    kill "$VAULT_PID" >/dev/null 2>&1 || true
    wait "$VAULT_PID" 2>/dev/null || true
    VAULT_PID=""
  fi
}

restart_vault() {
  echo "Restarting Vault so persisted plugin mounts reload the current catalog..."
  stop_vault
  start_vault
  wait_for_vault
  unseal_vault
}

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
  fi
}

unseal_vault() {
  if vault status | grep -q "Sealed.*true"; then
    UNSEAL_KEYS=$(jq -r '.unseal_keys_b64[]' "$INIT_FILE" 2>/dev/null)
    if [ -z "$UNSEAL_KEYS" ]; then
      echo "Warning: No unseal keys found. Cannot unseal Vault."
      return
    fi
    for key in $UNSEAL_KEYS; do
      vault operator unseal "$key" >/dev/null 2>&1 || true
    done
  else
    echo "Vault already unsealed."
  fi
}

login_root() {
  export VAULT_TOKEN=$(jq -r '.root_token' "$INIT_FILE")
}

register_plugin() {
  echo "Registering plugin $PLUGIN_NAME with checksum $PLUGIN_SHA256..."
  vault plugin register -sha256="$PLUGIN_SHA256" secret "$PLUGIN_NAME"
}

plugin_mount_healthy() {
  vault path-help "$PLUGIN_PATH/wallets/tltc" >/dev/null 2>&1
}

disable_plugin_mount() {
  i=1
  while [ "$i" -le 15 ]; do
    if vault secrets disable "$PLUGIN_PATH"; then
      return 0
    fi
    echo "Waiting for Vault to allow mount changes..."
    sleep 1
    i=$((i + 1))
  done

  return 1
}

enable_plugin_mount() {
  i=1
  while [ "$i" -le 15 ]; do
    if vault secrets enable -plugin-name="$PLUGIN_NAME" -path="$PLUGIN_PATH" plugin; then
      return 0
    fi
    echo "Waiting for Vault to enable plugin mount..."
    sleep 1
    i=$((i + 1))
  done

  return 1
}

enable_plugin() {
  MOUNT_INFO=$(vault secrets list -detailed -format=json | jq -r --arg path "$PLUGIN_PATH/" '.[$path] // empty')
  if [ -n "$MOUNT_INFO" ]; then
    RUNNING_SHA=$(printf '%s' "$MOUNT_INFO" | jq -r '.running_sha256 // ""')
    if [ "$RUNNING_SHA" = "$PLUGIN_SHA256" ] && plugin_mount_healthy; then
      echo "Plugin already enabled at $PLUGIN_PATH/ with current checksum."
      return
    fi

    if [ "$RUNNING_SHA" != "$PLUGIN_SHA256" ]; then
      echo "Plugin mount has stale checksum ${RUNNING_SHA:-none}; reloading Vault after catalog update..."
      restart_vault
      MOUNT_INFO=$(vault secrets list -detailed -format=json | jq -r --arg path "$PLUGIN_PATH/" '.[$path] // empty')
      RUNNING_SHA=$(printf '%s' "$MOUNT_INFO" | jq -r '.running_sha256 // ""')
      if [ "$RUNNING_SHA" = "$PLUGIN_SHA256" ] && plugin_mount_healthy; then
        echo "Plugin mount reloaded at $PLUGIN_PATH/ with current checksum."
        return
      fi
    fi

    echo "Disabling stale plugin mount $PLUGIN_PATH/ (running checksum: ${RUNNING_SHA:-none})..."
    disable_plugin_mount || return 1
  fi

  echo "Enabling plugin at $PLUGIN_PATH/..."
  enable_plugin_mount || return 1
}

main() {
  wait_for_vault
  init_vault
  unseal_vault
  login_root
  register_plugin
  enable_plugin
  echo "Vault setup complete."
}

start_vault

set +e
main
SETUP_STATUS=$?
set -e

if [ "$SETUP_STATUS" -ne 0 ]; then
  echo "Warning: Vault bootstrap steps failed, keeping server running."
fi

wait "$VAULT_PID"
