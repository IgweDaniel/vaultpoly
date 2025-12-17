package vaultpoly

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/igwedaniel/vaultpoly/internal/adapters"
)

func pathSign(b *pluginBackend) []*framework.Path {
	return []*framework.Path{
		{
			Pattern:      "wallets/" + framework.GenericNameRegex("blockchainType") + "/" + framework.GenericNameRegex("address") + "/sign",
			HelpSynopsis: "Sign a transaction using a wallet maintained by the plugin backend.",
			HelpDescription: `
	POST - sign a transaction for a given blockchain type

`,
			Fields: map[string]*framework.FieldSchema{
				"blockchainType": {
					Type:          framework.TypeString,
					Required:      true,
					Description:   "The blockchain type for the account. Currently supported: 'eth', 'btc', 'tbtc'.",
					AllowedValues: adapters.AllowedBlockchains(),
				},
				"address": {
					Type:        framework.TypeString,
					Required:    true,
					Description: "The address of the wallet to sign the transaction.",
				},
				"payload": {
					Type:        framework.TypeString,
					Required:    true,
					Description: "The txn payload to sign.",
				},
			},

			Callbacks: map[logical.Operation]framework.OperationFunc{
				logical.UpdateOperation: b.signTxn,
			},
		},
	}
}

func (b *pluginBackend) signTxn(ctx context.Context, req *logical.Request, d *framework.FieldData) (*logical.Response, error) {

	jsonPayload := d.Get("payload").(string)
	if jsonPayload == "" {
		return nil, logical.CodedError(http.StatusBadRequest, "payload is required")
	}

	blockchainType := adapters.BlockchainType(d.Get("blockchainType").(string))
	if !blockchainType.IsValid() {
		return nil, fmt.Errorf("invalid blockchain type: %s", blockchainType)
	}

	adapter, err := adapters.GetAdapter(blockchainType)
	if err != nil {
		return nil, err
	}

	walletAddress := d.Get("address").(string)
	if walletAddress == "" {
		return nil, fmt.Errorf("wallet address is required")
	}
	walletPath := fmt.Sprintf("wallets/%s/%s", blockchainType, walletAddress)
	entry, err := req.Storage.Get(ctx, walletPath)
	if err != nil {
		b.Logger().Error("Failed to retrieve the account by address", "path", walletPath, "error", err)
		return nil, err
	}
	if entry == nil {
		return nil, logical.CodedError(http.StatusExpectationFailed, fmt.Sprintf("no account found for address: %s", walletAddress))
	}
	var wallet adapters.Wallet
	_ = entry.DecodeJSON(&wallet)

	signature, err := adapter.CreateSignedTransaction(&wallet, jsonPayload)
	if err != nil {
		if err == adapters.ErrInvalidPayload {
			return nil, logical.CodedError(http.StatusBadRequest, "invalid payload format")
		}
		return nil, err
	}

	return &logical.Response{
		Data: map[string]interface{}{
			"signature": signature,
		},
	}, nil
}
