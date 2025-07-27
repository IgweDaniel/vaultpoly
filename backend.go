package vaultpoly

import (
	"context"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

func Factory(ctx context.Context, conf *logical.BackendConfig) (logical.Backend, error) {
	b := backend()
	if err := b.Setup(ctx, conf); err != nil {
		return nil, err
	}
	return b, nil
}

type pluginBackend struct {
	*framework.Backend
	// lock     sync.RWMutex
	// registry map[adapters.BlockchainType]adapters.BlockchainAdapter // registry for blockchain adapters
}

// backend defines the target API backend
// for Vault. It must include each path
// and the secrets it will store.
func backend() *pluginBackend {
	var b = pluginBackend{}
	// b.registry = make(map[adapters.BlockchainType]adapters.BlockchainAdapter)
	// b.registry[adapters.BlockchainETH] = eth.NewAdapter() // Assuming eth package implements

	b.Backend = &framework.Backend{
		Help: "",
		PathsSpecial: &logical.Paths{
			SealWrapStorage: []string{
				"accounts/",
			},
		},
		Paths: framework.PathAppend(
			walletsPaths(&b),
			pathSign(&b),
		),
		Secrets:     []*framework.Secret{},
		BackendType: logical.TypeLogical,
		// Invalidate:  b.invalidate,
	}
	return &b
}
