package adapters

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/base58"
)

type SolanaPayload struct {
	TransactionBase64 string `json:"transaction_base64"`
	TransactionBase58 string `json:"transaction_base58"`
	TransactionHex    string `json:"transaction_hex"`
	OutputEncoding    string `json:"output_encoding"`
}

type solanaAdapter struct{}

func NewSolanaAdapter() *solanaAdapter {
	return &solanaAdapter{}
}

func (a *solanaAdapter) ImportWallet(privateKeyValue string) (*Wallet, error) {
	privateKey, err := solanaPrivateKeyFromString(privateKeyValue)
	if err != nil {
		return nil, err
	}

	return &Wallet{
		PrivateKey: privateKey.String(),
		PublicKey:  privateKey.PublicKey().String(),
	}, nil
}

func (a *solanaAdapter) DeriveWallet() (*Wallet, error) {
	privateKey, err := solana.NewRandomPrivateKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate Solana private key: %w", err)
	}

	return &Wallet{
		PrivateKey: privateKey.String(),
		PublicKey:  privateKey.PublicKey().String(),
	}, nil
}

func (a *solanaAdapter) validatePayload(jsonPayload string) (*SolanaPayload, error) {
	var payload SolanaPayload
	if err := json.Unmarshal([]byte(jsonPayload), &payload); err != nil {
		return nil, fmt.Errorf("failed to decode payload: %w", err)
	}

	if payload.TransactionBase64 == "" && payload.TransactionBase58 == "" && payload.TransactionHex == "" {
		return nil, fmt.Errorf("payload must contain transaction_base64, transaction_base58, or transaction_hex")
	}

	if payload.OutputEncoding == "" {
		payload.OutputEncoding = "base64"
	}

	switch payload.OutputEncoding {
	case "base64", "base58", "hex":
	default:
		return nil, fmt.Errorf("unsupported output_encoding: %s", payload.OutputEncoding)
	}

	return &payload, nil
}

func (a *solanaAdapter) CreateSignedTransaction(wallet *Wallet, payload string) (string, error) {
	solanaPayload, err := a.validatePayload(payload)
	if err != nil {
		return "", err
	}

	privateKey, err := solana.PrivateKeyFromBase58(strings.TrimSpace(wallet.PrivateKey))
	if err != nil {
		return "", fmt.Errorf("failed to decode Solana private key: %w", err)
	}

	tx, err := solanaPayload.transaction()
	if err != nil {
		return "", err
	}

	_, err = tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		if key.Equals(privateKey.PublicKey()) {
			return &privateKey
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("failed to sign Solana transaction: %w", err)
	}

	return encodeSolanaTransaction(tx, solanaPayload.OutputEncoding)
}

func (p *SolanaPayload) transaction() (*solana.Transaction, error) {
	switch {
	case p.TransactionBase64 != "":
		tx, err := solana.TransactionFromBase64(strings.TrimSpace(p.TransactionBase64))
		if err != nil {
			return nil, fmt.Errorf("failed to decode transaction_base64: %w", err)
		}
		return tx, nil
	case p.TransactionBase58 != "":
		tx, err := solana.TransactionFromBase58(strings.TrimSpace(p.TransactionBase58))
		if err != nil {
			return nil, fmt.Errorf("failed to decode transaction_base58: %w", err)
		}
		return tx, nil
	case p.TransactionHex != "":
		rawTx, err := hex.DecodeString(normalizeHex(p.TransactionHex))
		if err != nil {
			return nil, fmt.Errorf("failed to decode transaction_hex: %w", err)
		}
		tx, err := solana.TransactionFromBytes(rawTx)
		if err != nil {
			return nil, fmt.Errorf("failed to decode transaction_hex bytes: %w", err)
		}
		return tx, nil
	default:
		return nil, fmt.Errorf("payload must contain a transaction")
	}
}

func encodeSolanaTransaction(tx *solana.Transaction, outputEncoding string) (string, error) {
	switch outputEncoding {
	case "base64":
		out, err := tx.ToBase64()
		if err != nil {
			return "", fmt.Errorf("failed to encode signed transaction as base64: %w", err)
		}
		return out, nil
	case "base58":
		rawTx, err := tx.MarshalBinary()
		if err != nil {
			return "", fmt.Errorf("failed to encode signed transaction bytes: %w", err)
		}
		return base58.Encode(rawTx), nil
	case "hex":
		rawTx, err := tx.MarshalBinary()
		if err != nil {
			return "", fmt.Errorf("failed to encode signed transaction bytes: %w", err)
		}
		return hex.EncodeToString(rawTx), nil
	default:
		return "", fmt.Errorf("unsupported output_encoding: %s", outputEncoding)
	}
}

func solanaPrivateKeyFromString(privateKeyValue string) (solana.PrivateKey, error) {
	trimmed := strings.TrimSpace(privateKeyValue)
	if trimmed == "" {
		return nil, fmt.Errorf("private key is required")
	}

	if strings.HasPrefix(trimmed, "[") {
		privateKey, err := solana.PrivateKeyFromSolanaKeygenFileBytes([]byte(trimmed))
		if err != nil {
			return nil, fmt.Errorf("invalid Solana keygen JSON private key: %w", err)
		}
		return privateKey, nil
	}

	privateKey, err := solana.PrivateKeyFromBase58(trimmed)
	if err != nil {
		return nil, fmt.Errorf("invalid Solana base58 private key: %w", err)
	}

	return privateKey, nil
}
