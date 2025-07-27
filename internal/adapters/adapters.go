package adapters

import (
	"fmt"
)

var ErrInvalidPayload = fmt.Errorf("invalid payload format")

type BlockchainAdapter interface {
	DeriveWallet() (*Wallet, error)
	CreateSignedTransaction(wallet *Wallet, payload string) (string, error)
}
