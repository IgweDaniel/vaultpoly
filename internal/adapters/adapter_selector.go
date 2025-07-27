package adapters

import (
	"fmt"
)

func GetAdapter(blockchainType BlockchainType) (BlockchainAdapter, error) {
	switch blockchainType {
	case BlockchainETH:
		return NewEthAdapter(), nil
	case BlockchainBTC:
		return NewBtcAdapter(), nil
	default:
		return nil, fmt.Errorf("unsupported blockchain type: %s", blockchainType)
	}
}
