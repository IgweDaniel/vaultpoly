package adapters

import (
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
)

type EthPayload struct {
	ChainID  uint64 `json:"chainId"`
	To       string `json:"to"`
	Value    uint64 `json:"value"`
	Data     string `json:"data"`
	GasLimit uint64 `json:"gas"`
	GasPrice uint64 `json:"gasPrice"`
	Nonce    uint64 `json:"nonce"`
}

type ethereumAdapter struct {
}

func NewEthAdapter() *ethereumAdapter {
	return &ethereumAdapter{}
}

func (a *ethereumAdapter) DeriveWallet() (*Wallet, error) {
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}
	privateKeyBytes := crypto.FromECDSA(privateKey)

	publicKey := privateKey.Public()

	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("failed to cast public key to ECDSA")

	}
	return &Wallet{
		PrivateKey: hexutil.Encode(privateKeyBytes)[2:],
		PublicKey:  crypto.PubkeyToAddress(*publicKeyECDSA).Hex(),
	}, nil
}

func (a *ethereumAdapter) validatePayload(jsonPayload string) (*EthPayload, error) {
	var payload EthPayload
	if err := json.Unmarshal([]byte(jsonPayload), &payload); err != nil {
		return nil, fmt.Errorf("failed to decode payload: %w", err)
	}
	if payload.To == "" {
		return nil, fmt.Errorf("payload must contain 'to' field")
	}

	if payload.GasLimit == 0 {
		payload.GasLimit = 21000 // Default gas limit for a simple transaction
	}
	if payload.GasPrice == 0 {
		payload.GasPrice = 20000000000 // Default gas price (20 Gwei)
	}
	return &payload, nil
}

func (a *ethereumAdapter) CreateSignedTransaction(wallet *Wallet, payload string) (string, error) {

	ethPayload, err := a.validatePayload(payload)
	if err != nil {
		return "", err
	}

	privateKey, err := crypto.HexToECDSA(wallet.PrivateKey)
	if err != nil {
		return "", fmt.Errorf("failed to convert private key: %w", err)
	}

	value := new(big.Int).SetUint64(ethPayload.Value)

	gasPrice := new(big.Int).SetUint64(ethPayload.GasPrice)

	// data is in hex, load hex as bytes
	data := make([]byte, 0)
	if ethPayload.Data != "" {
		data, err = hex.DecodeString(ethPayload.Data[2:]) // Remove '0x' prefix
		if err != nil {
			return "", fmt.Errorf("failed to decode data: %w", err)
		}

	}
	tx := types.NewTransaction(ethPayload.Nonce, common.HexToAddress(ethPayload.To), value, ethPayload.GasLimit, gasPrice, data)

	chainID := new(big.Int).SetUint64(ethPayload.ChainID)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign transaction: %w", err)
	}
	rawTxBytes, err := rlp.EncodeToBytes(signedTx)
	if err != nil {
		return "", fmt.Errorf("failed to encode transaction: %w", err)
	}
	rawTxHex := hex.EncodeToString(rawTxBytes)

	return rawTxHex, nil
}
