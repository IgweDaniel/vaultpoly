package vaultpoly

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"strings"
	"testing"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/ecdsa"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/igwedaniel/vaultpoly/internal/adapters"
	"github.com/stretchr/testify/require"
)

// for test wallet sign, we are going to have to have expected authorizer

func TestWalletSign(t *testing.T) {
	b, s := getTestBackend(t)

	t.Run("Sign Wallet ETH - pass", func(t *testing.T) {

		payload := adapters.EthPayload{
			ChainID:  97,
			To:       "0x337610d27c682E347C9cD60BD4b3b107C9d34dDd",
			Value:    0,
			Data:     "0xa9059cbb000000000000000000000000253f9dd15f4bd360595b0e83d51ef31d8e71d31b0000000000000000000000000000000000000000000000000de0b6b3a7640000",
			Nonce:    0,
			GasLimit: 60000,
			GasPrice: 1000000000, // 20 Gwei
		}
		jsonB, _ := json.Marshal(payload)

		resp, err := testWalletCreate(t, b, s,
			adapters.BlockchainETH.String(),
			map[string]interface{}{})
		require.NoError(t, err)
		require.NotEmpty(t, resp.Data["address"])

		address := resp.Data["address"].(string)
		resp, err = testWalletSign(t, b, s, adapters.BlockchainETH.String(), address, map[string]interface{}{
			"payload": string(jsonB),
		})

		signature := resp.Data["signature"].(string)
		require.Nil(t, err)
		require.Nil(t, resp.Error())
		require.NotNil(t, resp)
		require.NotEmpty(t, signature)

		txBytes, err := hex.DecodeString(strings.TrimPrefix(signature, "0x"))
		require.NoError(t, err, "Failed to decode transaction hex")
		var tx types.Transaction
		err = rlp.DecodeBytes(txBytes, &tx)

		require.NoError(t, err, "Failed to decode RLP transaction")
		require.NotNil(t, tx.To(), "Transaction should have a recipient")
		require.True(t, tx.Value().Cmp(big.NewInt(0)) >= 0, "Transaction value should be non-negative")
		require.True(t, tx.Gas() > 0, "Transaction should have gas limit")
		require.True(t, tx.GasPrice().Cmp(big.NewInt(0)) > 0, "Transaction should have gas price")
		chainID := big.NewInt(int64(payload.ChainID))

		signer := types.NewEIP155Signer(chainID)
		recoveredAddress, err := types.Sender(signer, &tx)
		require.NoError(t, err, "Failed to recover signer address")
		require.Equal(t, strings.ToLower(address), strings.ToLower(recoveredAddress.Hex()),
			"Recovered signer address doesn't match wallet address")
		require.Equal(t, strings.ToLower(payload.To), strings.ToLower(tx.To().Hex()),
			"Transaction recipient doesn't match payload")
		expectedValue := big.NewInt(int64(payload.Value))
		require.Equal(t, expectedValue.Cmp(tx.Value()), 0,
			"Transaction value doesn't match payload")
		require.Equal(t, uint64(payload.GasLimit), tx.Gas(),
			"Transaction gas limit doesn't match payload")
		expectedGasPrice := big.NewInt(int64(payload.GasPrice))
		require.Equal(t, expectedGasPrice.Cmp(tx.GasPrice()), 0,
			"Transaction gas price doesn't match payload")
		require.Equal(t, expectedGasPrice.Cmp(tx.GasPrice()), 0,
			"Transaction gas price doesn't match payload")
		require.Equal(t, uint64(payload.Nonce), tx.Nonce(),
			"Transaction nonce doesn't match payload")
		require.NoError(t, err, "Failed to decode expected data")
		require.Equal(t, strings.TrimPrefix(payload.Data, "0x"), hex.EncodeToString(tx.Data()),
			"Transaction data doesn't match payload")
	})

	t.Run("Sign Wallet BTC- pass", func(t *testing.T) {

		resp, err := testWalletCreate(t, b, s,
			adapters.BlockchainBTCTestnet.String(),
			map[string]interface{}{})

		address := resp.Data["address"].(string)
		require.NoError(t, err)
		require.NotEmpty(t, address)

		addr, err := btcutil.DecodeAddress(address, &chaincfg.TestNet4Params)
		require.NoError(t, err, "Failed to decode address")
		script, err := txscript.PayToAddrScript(addr)
		require.NoError(t, err, "Failed to create script")

		addressPubScriptKey := hex.EncodeToString(script)
		utxos := []adapters.UTXO{
			{
				Txid:             "9404a6b8f40b9fd4b868b0305a16eddfd1bcd8477c2f71bbc1588ba8884208c3",
				Vout:             1,
				Value:            500000,
				ScriptPubKey:     addressPubScriptKey,
				ScriptPubKeyType: "v0_p2wpkh",
			},
		}

		amount := int64(200000)                                   // 0.002 BTC
		recipient := "tb1qpn5dddjnc2qwurpsm449l6uvggnjxwsetrnksx" // Testnet address
		payload := testBtcPayload(amount, recipient, utxos)

		resp, err = testWalletSign(t, b, s, adapters.BlockchainBTCTestnet.String(), address, map[string]interface{}{
			"payload": payload,
		})

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Nil(t, resp.Error())
		require.NotEmpty(t, resp.Data)

		signature := resp.Data["signature"].(string)
		require.NotEmpty(t, signature)

		txBytes, err := hex.DecodeString(signature)
		require.NoError(t, err)

		var tx wire.MsgTx
		if err := tx.Deserialize(bytes.NewReader(txBytes)); err != nil {
			t.Errorf("Transaction is invalid: %v", err)
		}

		if len(tx.TxOut) != 2 {
			t.Errorf("Expected 2 outputs, got %d", len(tx.TxOut))
		}

		recipientAddr, _ := btcutil.DecodeAddress(recipient, &chaincfg.TestNet4Params)
		recipientScript, _ := txscript.PayToAddrScript(recipientAddr)
		if !bytes.Equal(tx.TxOut[0].PkScript, recipientScript) {
			t.Errorf("Recipient script mismatch: got %x, want %x", tx.TxOut[0].PkScript, recipientScript)
		}
		if tx.TxOut[0].Value != amount {
			t.Errorf("Recipient amount mismatch: got %d, want %d", tx.TxOut[0].Value, amount)
		}

		utxo := utxos[0]
		prevScript, err := hex.DecodeString(utxo.ScriptPubKey)
		require.NoError(t, err, "Failed to decode previous script")

		// Validate P2WPKH signature
		inputIndex := 0
		txIn := tx.TxIn[inputIndex]

		// Check that witness data exists (P2WPKH should have witness)
		require.NotEmpty(t, txIn.Witness, "P2WPKH transaction should have witness data")
		require.Equal(t, 2, len(txIn.Witness), "P2WPKH witness should have 2 elements (signature + pubkey)")

		// Extract signature and public key from witness
		sigBytes := txIn.Witness[0]
		pubKeyBytes := txIn.Witness[1]

		// Parse the signature (remove sighash type byte)
		if len(sigBytes) == 0 {
			t.Fatal("Empty signature in witness")
		}
		sigBytesNoHashType := sigBytes[:len(sigBytes)-1]
		signature_parsed, err := ecdsa.ParseDERSignature(sigBytesNoHashType)
		require.NoError(t, err, "Failed to parse signature")

		// Parse the public key
		pubKey, err := btcec.ParsePubKey(pubKeyBytes)
		require.NoError(t, err, "Failed to parse public key")

		// // Verify that the public key corresponds to the wallet address
		hash := btcutil.Hash160(pubKey.SerializeCompressed())
		derivedAddr, err := btcutil.NewAddressWitnessPubKeyHash(hash, &chaincfg.TestNet4Params)
		require.NoError(t, err, "Failed to derive address from public key")
		require.Equal(t, address, derivedAddr.EncodeAddress(), "Public key doesn't match wallet address")

		// // Verify the signature against the transaction
		sigHashes := txscript.NewTxSigHashes(&tx, txscript.NewCannedPrevOutputFetcher(prevScript, utxo.Value))
		sigHash, err := txscript.CalcWitnessSigHash(prevScript, sigHashes, txscript.SigHashAll, &tx, inputIndex, utxo.Value)
		require.NoError(t, err, "Failed to calculate signature hash")

		// // Verify the signature
		isValid := signature_parsed.Verify(sigHash, pubKey)
		require.True(t, isValid, "Signature verification failed - transaction was not properly signed by the wallet")
		prevOutputFetcher := txscript.NewCannedPrevOutputFetcher(prevScript, utxo.Value)
		// Additional validation using script engine for comprehensive check
		engine, err := txscript.NewEngine(
			prevScript,
			&tx,
			inputIndex,
			txscript.StandardVerifyFlags,
			nil,
			nil,
			utxo.Value,
			prevOutputFetcher,
		)
		require.NoError(t, err, "Failed to create script engine")

		err = engine.Execute()
		require.NoError(t, err, "Script execution failed - transaction signature is invalid")

		// Validate fee calculation
		totalInput := int64(utxo.Value)
		totalOutput := int64(0)
		for _, out := range tx.TxOut {
			totalOutput += out.Value
		}
		actualFee := totalInput - totalOutput
		require.True(t, actualFee > 0, "Transaction fee should be positive")
		require.True(t, actualFee < totalInput/2, "Transaction fee seems unreasonably high")
	})

}

func testWalletSign(t *testing.T, b *pluginBackend, s logical.Storage, blockchainType, address string, d map[string]interface{}) (*logical.Response, error) {
	t.Helper()
	resp, err := b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.UpdateOperation,
		Path:      "wallets/" + blockchainType + "/" + address + "/sign",
		Data:      d,
		Storage:   s,
	})

	if err != nil {
		return nil, err
	}

	return resp, nil
}

func testBtcPayload(amount int64, recipient string, utxos []adapters.UTXO) string {

	jsonB, _ := json.Marshal(adapters.BtcPayload{
		Utxos:     utxos,
		Recipient: recipient,
		Amount:    amount,
		FeeRate:   1.0, // 1 sat/vbyte
	})
	return string(jsonB)
}
