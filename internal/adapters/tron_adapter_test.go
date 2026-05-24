package adapters

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"github.com/btcsuite/btcd/btcutil/base58"
	"github.com/ethereum/go-ethereum/crypto"
)

const tronNileUSDTTransferRawDataHex = "0a0284122208d4c3ce31ed0ef92c40e0d5858fe5335aae01081f12a9010a31747970652e676f6f676c65617069732e636f6d2f70726f746f636f6c2e54726967676572536d617274436f6e747261637412740a154166f0494d0de9f4fae19b118d1e0ae7019cff5f29121541eca9bc828a3005b9a3b909f2cc5c2a54794de05f2244a9059cbb0000000000000000000000009e7bf6be6815ad07a306b80934963397709e787d00000000000000000000000000000000000000000000000000000000000f4240708081828fe5339001c0f4a46b"
const tronNileUSDTTransferTxID = "ec884c553cd14bc3917b8e79e0677d7b9e5e95a0f96ba3bca4cfa4cc30473a35"

func TestTronAdapterDeriveWallet(t *testing.T) {
	a := NewTronAdapter()

	wallet, err := a.DeriveWallet()
	if err != nil {
		t.Fatal(err)
	}

	if !strings.HasPrefix(wallet.PublicKey, "T") {
		t.Fatalf("expected TRON address to start with T, got %s", wallet.PublicKey)
	}

	payload, version, err := base58.CheckDecode(wallet.PublicKey)
	if err != nil {
		t.Fatalf("invalid TRON base58 address: %v", err)
	}
	if version != 0x41 {
		t.Fatalf("expected TRON address version 0x41, got 0x%x", version)
	}
	if len(payload) != 20 {
		t.Fatalf("expected TRON address payload length 20, got %d", len(payload))
	}
	if len(wallet.PrivateKey) != 64 {
		t.Fatalf("expected 32-byte private key hex, got length %d", len(wallet.PrivateKey))
	}
}

func TestTronAdapterImportWalletKnownPrivateKey(t *testing.T) {
	a := NewTronAdapter()

	privateKey := "aa18efe8e8b1da4488e9f1350ad1f25ad387229177907ddb5c57e9cc22a74592"
	expectedAddress := "TCXxPfJ15HiAAwEeHmAAg6cNKdAsXk4abF"

	wallet, err := a.ImportWallet(privateKey)
	if err != nil {
		t.Fatal(err)
	}

	if wallet.PublicKey != expectedAddress {
		t.Fatalf("unexpected TRON address. got=%s want=%s", wallet.PublicKey, expectedAddress)
	}
	if wallet.PrivateKey != privateKey {
		t.Fatalf("unexpected private key normalization. got=%s want=%s", wallet.PrivateKey, privateKey)
	}
}

func TestTronAdapterCreateSignedTransactionRawDataHex(t *testing.T) {
	a := NewTronAdapter()

	wallet, err := a.ImportWallet("aa18efe8e8b1da4488e9f1350ad1f25ad387229177907ddb5c57e9cc22a74592")
	if err != nil {
		t.Fatal(err)
	}

	rawDataHex := tronNileUSDTTransferRawDataHex
	payloadBytes, err := json.Marshal(TronPayload{RawDataHex: rawDataHex})
	if err != nil {
		t.Fatal(err)
	}

	sigHex, err := a.CreateSignedTransaction(wallet, string(payloadBytes))
	if err != nil {
		t.Fatal(err)
	}

	sigBytes, err := hex.DecodeString(sigHex)
	if err != nil {
		t.Fatalf("signature is not hex: %v", err)
	}
	if len(sigBytes) != 65 {
		t.Fatalf("expected 65-byte signature, got %d", len(sigBytes))
	}

	rawDataBytes, err := hex.DecodeString(rawDataHex)
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(rawDataBytes)

	recoveredPub, err := crypto.SigToPub(hash[:], sigBytes)
	if err != nil {
		t.Fatalf("failed to recover signer public key: %v", err)
	}

	recoveredAddress, err := tronAddressFromPubKey(recoveredPub)
	if err != nil {
		t.Fatalf("failed to derive TRON address from recovered pubkey: %v", err)
	}

	if recoveredAddress != wallet.PublicKey {
		t.Fatalf("recovered signer address mismatch. got=%s want=%s", recoveredAddress, wallet.PublicKey)
	}
}

func TestTronAdapterCreateSignedTransactionWithTransaction(t *testing.T) {
	a := NewTronAdapter()

	wallet, err := a.ImportWallet("aa18efe8e8b1da4488e9f1350ad1f25ad387229177907ddb5c57e9cc22a74592")
	if err != nil {
		t.Fatal(err)
	}

	rawDataHex := tronNileUSDTTransferRawDataHex
	payloadBytes, err := json.Marshal(TronPayload{
		RawDataHex: rawDataHex,
		Transaction: map[string]interface{}{
			"txID":         tronNileUSDTTransferTxID,
			"raw_data_hex": rawDataHex,
			"raw_data": map[string]interface{}{
				"contract": []interface{}{
					map[string]interface{}{
						"type": "TriggerSmartContract",
						"parameter": map[string]interface{}{
							"type_url": "type.googleapis.com/protocol.TriggerSmartContract",
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	signedTxJSON, err := a.CreateSignedTransaction(wallet, string(payloadBytes))
	if err != nil {
		t.Fatal(err)
	}

	var signedTx map[string]interface{}
	if err := json.Unmarshal([]byte(signedTxJSON), &signedTx); err != nil {
		t.Fatalf("signed transaction response must be JSON: %v", err)
	}

	signatures, ok := signedTx["signature"].([]interface{})
	if !ok {
		t.Fatalf("signed transaction missing signature array: %#v", signedTx["signature"])
	}
	if len(signatures) != 1 {
		t.Fatalf("expected one signature, got %d", len(signatures))
	}

	sigHex, ok := signatures[0].(string)
	if !ok {
		t.Fatalf("signature entry must be string, got %#v", signatures[0])
	}

	sigBytes, err := hex.DecodeString(sigHex)
	if err != nil {
		t.Fatalf("signature entry is not hex: %v", err)
	}
	if len(sigBytes) != 65 {
		t.Fatalf("expected signature length 65, got %d", len(sigBytes))
	}

	if signedTx["txID"] != tronNileUSDTTransferTxID {
		t.Fatalf("expected txID to be preserved, got %#v", signedTx["txID"])
	}
}
