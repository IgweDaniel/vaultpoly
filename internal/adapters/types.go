package adapters

type BlockchainType string

type Wallet struct {
	PublicKey  string `json:"public_key"`
	PrivateKey string `json:"private_key"`
}

const (
	BlockchainETH BlockchainType = "eth"
	BlockchainBTC BlockchainType = "btc"
)

var SupportedBlockchains = []BlockchainType{
	BlockchainETH,
	BlockchainBTC,
}

// validate
func (bt BlockchainType) IsValid() bool {
	for _, validType := range SupportedBlockchains {
		if bt == validType {
			return true
		}
	}
	return false
}

func (bt BlockchainType) String() string {
	return string(bt)
}
