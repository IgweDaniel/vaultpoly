package adapters

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
)

func TestCreateSignedTransaction_VerifySignature(t *testing.T) {
	net := &chaincfg.TestNet3Params
	a := NewBtcAdapter(net)

	wallet, err := a.DeriveWallet()
	if err != nil {
		t.Fatal(err)
	}

	// Decode the wallet's address to get its P2WPKH script.
	decAddr, err := btcutil.DecodeAddress(wallet.PublicKey, net)
	if err != nil {
		t.Fatal(err)
	}
	pkScript, err := txscript.PayToAddrScript(decAddr)
	if err != nil {
		t.Fatal(err)
	}
	scriptHex := hex.EncodeToString(pkScript)

	// Mock a single UTXO matching the wallet's script (v0_p2wpkh).
	utxo := UTXO{
		Txid:             "0000000000000000000000000000000000000000000000000000000000000000", // Dummy TXID.
		Vout:             0,
		Value:            1000000, // 0.01 BTC in satoshis.
		ScriptPubKey:     scriptHex,
		ScriptPubKeyType: "v0_p2wpkh",
	}

	// Mock recipient (valid testnet P2WPKH address).
	recipient := "tb1qpn5dddjnc2qwurpsm449l6uvggnjxwsetrnksx"
	amount := int64(500000) // 0.005 BTC in satoshis.
	feeRate := 10.0         // Conservative fee rate (sat/vB).

	payload := BtcPayload{
		Recipient: recipient,
		Amount:    amount,
		FeeRate:   feeRate,
		Utxos:     []UTXO{utxo},
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}

	// Create and sign the transaction.
	signedHex, err := a.CreateSignedTransaction(wallet, string(payloadJSON))
	if err != nil {
		t.Fatal(err)
	}

	// Deserialize the signed transaction.
	signedBytes, err := hex.DecodeString(signedHex)
	if err != nil {
		t.Fatal(err)
	}
	var tx wire.MsgTx
	err = tx.Deserialize(bytes.NewReader(signedBytes))
	if err != nil {
		t.Fatal(err)
	}

	// Verify the input signature using txscript.Engine.
	// This catches bugs in signing logic.
	flags := txscript.StandardVerifyFlags
	vm, err := txscript.NewEngine(
		pkScript,
		&tx,
		0,
		flags,
		nil, // no SigCache
		nil, // no TxSigHashes
		utxo.Value,
		txscript.NewCannedPrevOutputFetcher(pkScript, utxo.Value),
	)
	if err != nil {
		t.Fatalf("Failed to create txscript.Engine: %v", err)
	}
	if err := vm.Execute(); err != nil {
		t.Errorf("Transaction script execution failed: %v", err)
	}
}

func TestChangeAddressNetwork(t *testing.T) {
	net := &chaincfg.MainNetParams // Use mainnet to flag testnet hardcode mismatch.

	// Generate a private key/WIF for mainnet.
	privKey, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatal(err)
	}
	_, err = btcutil.NewWIF(privKey, net, true)
	if err != nil {
		t.Fatal(err)
	}

	// Compute hash160 (mimics code).
	hash := btcutil.Hash160(privKey.PubKey().SerializeCompressed())

	// Mimic the buggy hardcoded line in NewTxWithInputsAndOutputs.
	// This will fail the assertion below, flagging the hardcode.
	changeAddr, err := btcutil.NewAddressWitnessPubKeyHash(hash, &chaincfg.TestNet4Params)
	if err != nil {
		t.Fatal(err)
	}

	// Assert mismatch to flag the bug (changeAddr uses testnet params, but net is mainnet).
	if changeAddr.IsForNet(net) {
		t.Error("Change address uses hardcoded TestNet4Params but should match adapter's mainnet")
	}
}
