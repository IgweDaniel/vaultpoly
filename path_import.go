package vaultpoly

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/igwedaniel/vaultpoly/internal/adapters"
)

func importPaths(b *pluginBackend) []*framework.Path {
	return []*framework.Path{
		{
			Pattern:      "wallets/" + framework.GenericNameRegex("blockchainType") + "/import",
			HelpSynopsis: "Import an existing wallet by providing a private key.",
			HelpDescription: `

    POST - import a wallet by providing its private key.
           For Ethereum: provide the hex-encoded private key (with or without 0x prefix).
           For Bitcoin/Litecoin: provide the WIF-encoded private key.
           For Solana: provide the base58 private key or Solana keygen JSON array.

`,
			Fields: map[string]*framework.FieldSchema{
				"blockchainType": {
					Type:          framework.TypeString,
					Default:       "eth",
					Description:   "The blockchain type for the wallet. Currently supported: 'eth', 'btc', 'tbtc', 'tron', 'ltc', 'tltc', 'sol'.",
					AllowedValues: adapters.AllowedBlockchains(),
				},
				"private_key": {
					Type:        framework.TypeString,
					Required:    true,
					Description: "The private key to import. Hex-encoded for Ethereum/TRON, WIF-encoded for Bitcoin/Litecoin, base58 or keygen JSON for Solana.",
				},
			},

			Callbacks: map[logical.Operation]framework.OperationFunc{
				logical.UpdateOperation: b.pathWalletImport,
			},
		},
	}
}

func (b *pluginBackend) pathWalletImport(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {
	blockchainType := adapters.BlockchainType(d.Get("blockchainType").(string))
	if !blockchainType.IsValid() {
		return nil, fmt.Errorf("invalid blockchain type: %s", blockchainType)
	}

	privateKey := d.Get("private_key").(string)
	if privateKey == "" {
		return nil, logical.CodedError(http.StatusBadRequest, "private_key is required")
	}

	adapter, err := adapters.GetAdapter(blockchainType)
	if err != nil {
		return nil, err
	}

	wallet, err := adapter.ImportWallet(privateKey)
	if err != nil {
		b.Logger().Error("Failed to import wallet", "error", err)
		return nil, logical.CodedError(http.StatusBadRequest, fmt.Sprintf("failed to import wallet: %s", err))
	}

	walletPath := fmt.Sprintf("wallets/%s/%s", blockchainType, wallet.PublicKey)

	existing, err := req.Storage.Get(ctx, walletPath)
	if err != nil {
		return nil, fmt.Errorf("failed to check for existing wallet: %w", err)
	}
	if existing != nil {
		return nil, logical.CodedError(http.StatusConflict, fmt.Sprintf("wallet already exists: %s", wallet.PublicKey))
	}

	entry, err := logical.StorageEntryJSON(walletPath, wallet)
	if err != nil {
		b.Logger().Error("Failed to create storage entry for imported wallet", "error", err)
		return nil, fmt.Errorf("failed to create storage entry for wallet: %w", err)
	}

	err = req.Storage.Put(ctx, entry)
	if err != nil {
		b.Logger().Error("Failed to save the imported wallet to storage", "error", err)
		return nil, err
	}

	return &logical.Response{
		Data: map[string]interface{}{
			"address": wallet.PublicKey,
		},
	}, nil
}
