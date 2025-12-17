package adapters

import (
	"fmt"

	"github.com/btcsuite/btcd/chaincfg"
)

func GetAdapter(blockchainType BlockchainType) (BlockchainAdapter, error) {
	switch blockchainType {
	case BlockchainETH:
		return NewEthAdapter(), nil
	case BlockchainBTC:
		return NewBtcAdapter(&chaincfg.MainNetParams), nil
	case BlockchainBTCTestnet:
		return NewBtcAdapter(&chaincfg.TestNet4Params), nil
	default:
		return nil, fmt.Errorf("unsupported blockchain type: %s", blockchainType)
	}
}
