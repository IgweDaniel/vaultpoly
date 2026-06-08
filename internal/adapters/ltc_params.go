package adapters

import (
	"errors"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/wire"
)

const (
	litecoinMainNet  wire.BitcoinNet = 0xdbb6c0fb
	litecoinTestNet4 wire.BitcoinNet = 0xf1c8d2fd
)

var (
	LitecoinMainNetParams = &chaincfg.Params{
		Name:        "litecoin-mainnet",
		Net:         litecoinMainNet,
		DefaultPort: "9333",

		TargetTimespan:           time.Hour * 24 * 7 / 2,
		TargetTimePerBlock:       time.Second * 150,
		RetargetAdjustmentFactor: 4,

		Bech32HRPSegwit: "ltc",

		PubKeyHashAddrID: 0x30,
		ScriptHashAddrID: 0x32,
		PrivateKeyID:     0xb0,

		HDPrivateKeyID: [4]byte{0x04, 0x88, 0xad, 0xe4},
		HDPublicKeyID:  [4]byte{0x04, 0x88, 0xb2, 0x1e},
		HDCoinType:     2,
	}

	LitecoinTestNetParams = &chaincfg.Params{
		Name:        "litecoin-testnet4",
		Net:         litecoinTestNet4,
		DefaultPort: "19335",

		TargetTimespan:           time.Hour * 24 * 7 / 2,
		TargetTimePerBlock:       time.Second * 150,
		RetargetAdjustmentFactor: 4,
		ReduceMinDifficulty:      true,
		MinDiffReductionTime:     time.Minute * 5,

		Bech32HRPSegwit: "tltc",

		PubKeyHashAddrID: 0x6f,
		ScriptHashAddrID: 0x3a,
		PrivateKeyID:     0xef,

		HDPrivateKeyID: [4]byte{0x04, 0x35, 0x83, 0x94},
		HDPublicKeyID:  [4]byte{0x04, 0x35, 0x87, 0xcf},
		HDCoinType:     1,
	}
)

func init() {
	registerLitecoinParams(LitecoinMainNetParams)
	registerLitecoinParams(LitecoinTestNetParams)
}

func registerLitecoinParams(params *chaincfg.Params) {
	if err := chaincfg.Register(params); err != nil && !errors.Is(err, chaincfg.ErrDuplicateNet) {
		panic(err)
	}
}
