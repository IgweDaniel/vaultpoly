package adapters

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/btcsuite/btcd/btcutil/base58"
	"github.com/ethereum/go-ethereum/crypto"
)

type TronPayload struct {
	RawDataHex  string                 `json:"raw_data_hex"`
	Transaction map[string]interface{} `json:"transaction,omitempty"`
}

type tronAdapter struct{}

func NewTronAdapter() *tronAdapter {
	return &tronAdapter{}
}

func (a *tronAdapter) ImportWallet(privateKeyHex string) (*Wallet, error) {
	privateKey, err := crypto.HexToECDSA(normalizeHex(privateKeyHex))
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}

	address, err := tronAddressFromPubKey(&privateKey.PublicKey)
	if err != nil {
		return nil, err
	}

	return &Wallet{
		PrivateKey: hex.EncodeToString(crypto.FromECDSA(privateKey)),
		PublicKey:  address,
	}, nil
}

func (a *tronAdapter) DeriveWallet() (*Wallet, error) {
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	address, err := tronAddressFromPubKey(&privateKey.PublicKey)
	if err != nil {
		return nil, err
	}

	return &Wallet{
		PrivateKey: hex.EncodeToString(crypto.FromECDSA(privateKey)),
		PublicKey:  address,
	}, nil
}

func (a *tronAdapter) validatePayload(jsonPayload string) (*TronPayload, error) {
	var payload TronPayload
	if err := json.Unmarshal([]byte(jsonPayload), &payload); err != nil {
		return nil, fmt.Errorf("failed to decode payload: %w", err)
	}

	if payload.RawDataHex == "" && payload.Transaction != nil {
		if txRawDataHex, ok := payload.Transaction["raw_data_hex"].(string); ok {
			payload.RawDataHex = txRawDataHex
		}
	}

	if normalizeHex(payload.RawDataHex) == "" {
		return nil, fmt.Errorf("payload must contain 'raw_data_hex'")
	}

	return &payload, nil
}

func (a *tronAdapter) CreateSignedTransaction(wallet *Wallet, payload string) (string, error) {
	tronPayload, err := a.validatePayload(payload)
	if err != nil {
		return "", err
	}

	privateKey, err := crypto.HexToECDSA(normalizeHex(wallet.PrivateKey))
	if err != nil {
		return "", fmt.Errorf("failed to convert private key: %w", err)
	}

	rawDataBytes, err := hex.DecodeString(normalizeHex(tronPayload.RawDataHex))
	if err != nil {
		return "", fmt.Errorf("failed to decode raw_data_hex: %w", err)
	}

	hash := sha256.Sum256(rawDataBytes)
	sigBytes, err := crypto.Sign(hash[:], privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign transaction: %w", err)
	}

	sigHex := hex.EncodeToString(sigBytes)
	if tronPayload.Transaction == nil {
		return sigHex, nil
	}

	signedTx := make(map[string]interface{}, len(tronPayload.Transaction)+1)
	for k, v := range tronPayload.Transaction {
		signedTx[k] = v
	}

	if existing, ok := signedTx["signature"]; ok {
		switch values := existing.(type) {
		case []interface{}:
			signedTx["signature"] = append(values, sigHex)
		default:
			signedTx["signature"] = []string{sigHex}
		}
	} else {
		signedTx["signature"] = []string{sigHex}
	}

	signedTxJSON, err := json.Marshal(signedTx)
	if err != nil {
		return "", fmt.Errorf("failed to encode signed transaction: %w", err)
	}

	return string(signedTxJSON), nil
}

func tronAddressFromPubKey(publicKey *ecdsa.PublicKey) (string, error) {
	if publicKey == nil {
		return "", fmt.Errorf("public key is required")
	}

	pubKeyBytes := crypto.FromECDSAPub(publicKey)
	if len(pubKeyBytes) == 0 {
		return "", fmt.Errorf("failed to encode public key")
	}

	hash := crypto.Keccak256(pubKeyBytes[1:])
	ethAddress := hash[len(hash)-20:]

	return base58.CheckEncode(ethAddress, 0x41), nil
}

func normalizeHex(value string) string {
	trimmed := strings.TrimSpace(value)
	if strings.HasPrefix(trimmed, "0x") || strings.HasPrefix(trimmed, "0X") {
		return trimmed[2:]
	}
	return trimmed
}
