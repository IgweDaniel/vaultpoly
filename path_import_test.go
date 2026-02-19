package vaultpoly

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"strings"
	"testing"

	btcec "github.com/btcsuite/btcd/btcec/v2"
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

func TestImportAndSign(t *testing.T) {
	b, s := getTestBackend(t)

	t.Run("Import and sign ETH", func(t *testing.T) {
		testImportAndSignETH(t, b, s)
	})

	t.Run("Import and sign BTC", func(t *testing.T) {
		testImportAndSignBTC(t, b, s)
	})
}

func testImportAndSignETH(t *testing.T, b *pluginBackend, s logical.Storage) {
	importResp, err := testWalletImport(t, b, s, adapters.BlockchainETH.String(),
		"aa18efe8e8b1da4488e9f1350ad1f25ad387229177907ddb5c57e9cc22a74592")
	require.NoError(t, err)
	require.NotNil(t, importResp)
	require.Nil(t, importResp.Error())

	address := importResp.Data["address"].(string)
	require.NotEmpty(t, address)

	payload := adapters.EthPayload{
		ChainID:  97,
		To:       "0x337610d27c682E347C9cD60BD4b3b107C9d34dDd",
		Value:    0,
		Data:     "0xa9059cbb000000000000000000000000253f9dd15f4bd360595b0e83d51ef31d8e71d31b0000000000000000000000000000000000000000000000000de0b6b3a7640000",
		Nonce:    0,
		GasLimit: 60000,
		GasPrice: 1000000000,
	}
	jsonB, _ := json.Marshal(payload)

	signResp, err := testWalletSign(t, b, s, adapters.BlockchainETH.String(), address, map[string]interface{}{
		"payload": string(jsonB),
	})
	require.NoError(t, err)
	require.NotNil(t, signResp)
	require.Nil(t, signResp.Error())

	signature := signResp.Data["signature"].(string)
	require.NotEmpty(t, signature)

	txBytes, err := hex.DecodeString(strings.TrimPrefix(signature, "0x"))
	require.NoError(t, err, "Failed to decode transaction hex")

	var tx types.Transaction
	err = rlp.DecodeBytes(txBytes, &tx)
	require.NoError(t, err, "Failed to decode RLP transaction")

	require.NotNil(t, tx.To(), "Transaction should have a recipient")
	require.Equal(t, uint64(payload.GasLimit), tx.Gas())
	require.Equal(t, uint64(payload.Nonce), tx.Nonce())
	require.Equal(t, strings.TrimPrefix(payload.Data, "0x"), hex.EncodeToString(tx.Data()))
	require.Equal(t, strings.ToLower(payload.To), strings.ToLower(tx.To().Hex()))

	expectedValue := big.NewInt(int64(payload.Value))
	require.Equal(t, 0, expectedValue.Cmp(tx.Value()))

	expectedGasPrice := big.NewInt(int64(payload.GasPrice))
	require.Equal(t, 0, expectedGasPrice.Cmp(tx.GasPrice()))

	chainID := big.NewInt(int64(payload.ChainID))
	signer := types.NewEIP155Signer(chainID)
	recoveredAddress, err := types.Sender(signer, &tx)
	require.NoError(t, err, "Failed to recover signer address")
	require.Equal(t, strings.ToLower(address), strings.ToLower(recoveredAddress.Hex()),
		"Recovered signer address doesn't match imported wallet address")
}

func testImportAndSignBTC(t *testing.T, b *pluginBackend, s logical.Storage) {
	pk, err := btcec.NewPrivateKey()
	require.NoError(t, err)
	wif, err := btcutil.NewWIF(pk, &chaincfg.TestNet4Params, true)
	require.NoError(t, err)

	importResp, err := testWalletImport(t, b, s, adapters.BlockchainBTCTestnet.String(), wif.String())
	require.NoError(t, err)
	require.NotNil(t, importResp)
	require.Nil(t, importResp.Error())

	address := importResp.Data["address"].(string)
	require.NotEmpty(t, address)

	addr, err := btcutil.DecodeAddress(address, &chaincfg.TestNet4Params)
	require.NoError(t, err)
	script, err := txscript.PayToAddrScript(addr)
	require.NoError(t, err)

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

	amount := int64(200000)
	recipient := "tb1qpn5dddjnc2qwurpsm449l6uvggnjxwsetrnksx"
	payload := testBtcPayload(amount, recipient, utxos)

	signResp, err := testWalletSign(t, b, s, adapters.BlockchainBTCTestnet.String(), address, map[string]interface{}{
		"payload": payload,
	})
	require.NoError(t, err)
	require.NotNil(t, signResp)
	require.Nil(t, signResp.Error())

	signature := signResp.Data["signature"].(string)
	require.NotEmpty(t, signature)

	txBytes, err := hex.DecodeString(signature)
	require.NoError(t, err)

	var tx wire.MsgTx
	err = tx.Deserialize(bytes.NewReader(txBytes))
	require.NoError(t, err, "Transaction should deserialize")
	require.Equal(t, 2, len(tx.TxOut), "Expected 2 outputs (recipient + change)")

	recipientAddr, err := btcutil.DecodeAddress(recipient, &chaincfg.TestNet4Params)
	require.NoError(t, err)
	recipientScript, err := txscript.PayToAddrScript(recipientAddr)
	require.NoError(t, err)
	require.True(t, bytes.Equal(tx.TxOut[0].PkScript, recipientScript), "Recipient script mismatch")
	require.Equal(t, amount, tx.TxOut[0].Value, "Recipient amount mismatch")

	utxo := utxos[0]
	prevScript, err := hex.DecodeString(utxo.ScriptPubKey)
	require.NoError(t, err)

	txIn := tx.TxIn[0]
	require.NotEmpty(t, txIn.Witness, "P2WPKH transaction should have witness data")
	require.Equal(t, 2, len(txIn.Witness), "P2WPKH witness should have 2 elements (signature + pubkey)")

	sigBytes := txIn.Witness[0]
	pubKeyBytes := txIn.Witness[1]

	sigBytesNoHashType := sigBytes[:len(sigBytes)-1]
	parsedSig, err := ecdsa.ParseDERSignature(sigBytesNoHashType)
	require.NoError(t, err, "Failed to parse signature")

	pubKey, err := btcec.ParsePubKey(pubKeyBytes)
	require.NoError(t, err, "Failed to parse public key")

	hash := btcutil.Hash160(pubKey.SerializeCompressed())
	derivedAddr, err := btcutil.NewAddressWitnessPubKeyHash(hash, &chaincfg.TestNet4Params)
	require.NoError(t, err)
	require.Equal(t, address, derivedAddr.EncodeAddress(),
		"Signing public key doesn't match imported wallet address")

	sigHashes := txscript.NewTxSigHashes(&tx, txscript.NewCannedPrevOutputFetcher(prevScript, utxo.Value))
	sigHash, err := txscript.CalcWitnessSigHash(prevScript, sigHashes, txscript.SigHashAll, &tx, 0, utxo.Value)
	require.NoError(t, err)
	require.True(t, parsedSig.Verify(sigHash, pubKey), "Signature verification failed")

	prevOutputFetcher := txscript.NewCannedPrevOutputFetcher(prevScript, utxo.Value)
	engine, err := txscript.NewEngine(prevScript, &tx, 0, txscript.StandardVerifyFlags, nil, nil, utxo.Value, prevOutputFetcher)
	require.NoError(t, err)
	err = engine.Execute()
	require.NoError(t, err, "Script execution failed - transaction signature is invalid")

	totalOutput := int64(0)
	for _, out := range tx.TxOut {
		totalOutput += out.Value
	}
	actualFee := utxo.Value - totalOutput
	require.True(t, actualFee > 0, "Transaction fee should be positive")
}

func testWalletImport(t *testing.T, b *pluginBackend, s logical.Storage, blockchainType string, privateKey string) (*logical.Response, error) {
	t.Helper()
	return b.HandleRequest(context.Background(), &logical.Request{
		Operation: logical.UpdateOperation,
		Path:      "wallets/" + blockchainType + "/import",
		Data: map[string]interface{}{
			"private_key": privateKey,
		},
		Storage: s,
	})
}
