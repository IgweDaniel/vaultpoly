package vaultpoly

import (
	"context"
	"fmt"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/igwedaniel/vaultpoly/internal/adapters"
)

const (
	Empty = ""
)

type Account struct {
	Address    string `json:"address"`
	PrivateKey string `json:"private_key"`
	PublicKey  string `json:"public_key"`
}

func walletsPaths(b *pluginBackend) []*framework.Path {
	return []*framework.Path{
		{
			Pattern:      "wallets/" + framework.GenericNameRegex("blockchainType") + "/?",
			HelpSynopsis: "List all the Wallets  maintained by the plugin backend and create new wallet for a blockchainType.",
			HelpDescription: `

    LIST - list all wallets for a given blockchain type
    POST - create a new account for a given blockchain type.

`,
			Fields: map[string]*framework.FieldSchema{
				"blockchainType": {
					Type:        framework.TypeString,
					Default:     "eth",
					Description: "The blockchain type for the account. Currently supported: 'eth', 'btc'.",
				},
				"mnemonic": {
					Type:        framework.TypeString,
					Default:     Empty,
					Description: "The mnemonic to use to create the account. If not provided, one is generated.",
				},
			},

			Callbacks: map[logical.Operation]framework.OperationFunc{
				logical.ListOperation:   b.listWallets,
				logical.UpdateOperation: b.pathAccountsCreate,
			},
		},
	}
}

func (b *pluginBackend) listWallets(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	vals, err := req.Storage.List(ctx, "wallets/"+d.Get("blockchainType").(string)+"/")
	if err != nil {
		b.Logger().Error("Failed to retrieve the list of accounts", "error", err)
		return nil, err
	}

	return logical.ListResponse(vals), nil
}

func (b *pluginBackend) pathAccountsCreate(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	blockchainType := adapters.BlockchainType(d.Get("blockchainType").(string))
	if !blockchainType.IsValid() {
		return nil, fmt.Errorf("invalid blockchain type: %s", blockchainType)
	}
	adapter, err := adapters.GetAdapter(blockchainType)
	if err != nil {
		return nil, err
	}

	wallet, err := adapter.DeriveWallet()
	if err != nil {
		b.Logger().Error("Failed to create wallet", "error", err)
		return nil, fmt.Errorf("failed to create wallet: %w", err)
	}

	walletPath := fmt.Sprintf("wallets/%s/%s", blockchainType, wallet.PublicKey)

	entry, err := logical.StorageEntryJSON(walletPath, wallet)
	if err != nil {
		b.Logger().Error("Failed to create storage entry for wallet", "error", err)
		return nil, fmt.Errorf("failed to create storage entry for wallet: %w", err)
	}

	err = req.Storage.Put(ctx, entry)
	if err != nil {
		b.Logger().Error("Failed to save the new account to storage", "error", err)
		return nil, err
	}

	return &logical.Response{
		Data: map[string]interface{}{
			"address": wallet.PublicKey,
		},
	}, nil
}
