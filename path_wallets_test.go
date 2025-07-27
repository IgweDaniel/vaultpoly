package vaultpoly

import (
	"context"
	"testing"

	"github.com/hashicorp/vault/sdk/logical"
	"github.com/igwedaniel/vaultpoly/internal/adapters"
	"github.com/stretchr/testify/require"
)

func TestWallets(t *testing.T) {
	b, s := getTestBackend(t)

	t.Run("List All Wallets", func(t *testing.T) {
		for i := 1; i <= 10; i++ {
			_, err := testWalletCreate(t, b, s,
				adapters.BlockchainETH.String(),
				map[string]interface{}{})
			require.NoError(t, err)
		}

		resp, err := testListWallets(t, b, s, adapters.BlockchainETH.String())
		require.NoError(t, err)
		require.Len(t, resp.Data["keys"].([]string), 10)
	})

	t.Run("Create Wallet - pass ETH", func(t *testing.T) {
		resp, err := testWalletCreate(t, b, s, adapters.BlockchainETH.String(), map[string]interface{}{})

		require.Nil(t, err)
		require.Nil(t, resp.Error())
		require.NotNil(t, resp)
		require.NotEmpty(t, resp.Data["address"])
	})

	t.Run("Create Wallet - pass BTC", func(t *testing.T) {
		resp, err := testWalletCreate(t, b, s, adapters.BlockchainBTC.String(), map[string]interface{}{})

		require.Nil(t, err)
		require.Nil(t, resp.Error())
		require.NotNil(t, resp)
		require.NotEmpty(t, resp.Data["address"])
	})

}

func testWalletCreate(t *testing.T, b *pluginBackend, s logical.Storage, blockchainType string, d map[string]interface{}) (*logical.Response, error) {
	t.Helper()
	resp, err := b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.UpdateOperation,
		Path:      "wallets/" + blockchainType,
		Data:      d,
		Storage:   s,
	})

	if err != nil {
		return nil, err
	}

	return resp, nil
}

func testListWallets(t *testing.T, b *pluginBackend, s logical.Storage, blockchainType string) (*logical.Response, error) {
	t.Helper()
	return b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.ListOperation,
		Path:      "wallets/" + blockchainType,
		Storage:   s,
	})
}
