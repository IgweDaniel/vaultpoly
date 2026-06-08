package adapters

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
)

func TestLitecoinWalletDerivationAndImport(t *testing.T) {
	a := NewBtcAdapter(LitecoinMainNetParams)

	wallet, err := a.DeriveWallet()
	if err != nil {
		t.Fatal(err)
	}

	if !strings.HasPrefix(wallet.PublicKey, "ltc1q") {
		t.Fatalf("expected native Litecoin address, got %s", wallet.PublicKey)
	}

	addr, err := btcutil.DecodeAddress(wallet.PublicKey, LitecoinMainNetParams)
	if err != nil {
		t.Fatal(err)
	}
	if !addr.IsForNet(LitecoinMainNetParams) {
		t.Fatalf("address %s is not for Litecoin mainnet", wallet.PublicKey)
	}

	wif, err := btcutil.DecodeWIF(wallet.PrivateKey)
	if err != nil {
		t.Fatal(err)
	}
	if !wif.IsForNet(LitecoinMainNetParams) {
		t.Fatal("WIF is not for Litecoin mainnet")
	}
	if wif.IsForNet(&chaincfg.MainNetParams) {
		t.Fatal("Litecoin mainnet WIF should not be accepted as Bitcoin mainnet")
	}

	imported, err := a.ImportWallet(wallet.PrivateKey)
	if err != nil {
		t.Fatal(err)
	}
	if imported.PublicKey != wallet.PublicKey {
		t.Fatalf("imported address mismatch. got=%s want=%s", imported.PublicKey, wallet.PublicKey)
	}

	_, err = NewBtcAdapter(&chaincfg.MainNetParams).ImportWallet(wallet.PrivateKey)
	if err == nil {
		t.Fatal("expected Bitcoin mainnet adapter to reject Litecoin mainnet WIF")
	}
}

func TestLitecoinNetworkParamsAddressPrefixes(t *testing.T) {
	if LitecoinTestNetParams.Name != "litecoin-testnet4" {
		t.Fatalf("expected Litecoin testnet4 params, got %s", LitecoinTestNetParams.Name)
	}
	if LitecoinTestNetParams.Net != litecoinTestNet4 {
		t.Fatalf("expected Litecoin testnet4 network magic, got %x", LitecoinTestNetParams.Net)
	}

	privateKeyBytes := bytes.Repeat([]byte{0x01}, btcec.PrivKeyBytesLen)
	privateKey, _ := btcec.PrivKeyFromBytes(privateKeyBytes)
	hash := btcutil.Hash160(privateKey.PubKey().SerializeCompressed())

	mainLegacy, err := btcutil.NewAddressPubKeyHash(hash, LitecoinMainNetParams)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(mainLegacy.EncodeAddress(), "L") {
		t.Fatalf("expected legacy Litecoin mainnet address to start with L, got %s", mainLegacy.EncodeAddress())
	}

	mainSegwit, err := btcutil.NewAddressWitnessPubKeyHash(hash, LitecoinMainNetParams)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(mainSegwit.EncodeAddress(), "ltc1q") {
		t.Fatalf("expected Litecoin mainnet bech32 address, got %s", mainSegwit.EncodeAddress())
	}

	testSegwit, err := btcutil.NewAddressWitnessPubKeyHash(hash, LitecoinTestNetParams)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(testSegwit.EncodeAddress(), "tltc1q") {
		t.Fatalf("expected Litecoin testnet bech32 address, got %s", testSegwit.EncodeAddress())
	}
}

func TestLitecoinCreateSignedTransaction_VerifySignature(t *testing.T) {
	a := NewBtcAdapter(LitecoinTestNetParams)

	wallet, err := a.DeriveWallet()
	if err != nil {
		t.Fatal(err)
	}

	decAddr, err := btcutil.DecodeAddress(wallet.PublicKey, LitecoinTestNetParams)
	if err != nil {
		t.Fatal(err)
	}
	pkScript, err := txscript.PayToAddrScript(decAddr)
	if err != nil {
		t.Fatal(err)
	}

	utxo := UTXO{
		Txid:             "0000000000000000000000000000000000000000000000000000000000000000",
		Vout:             0,
		Value:            1000000,
		ScriptPubKey:     hex.EncodeToString(pkScript),
		ScriptPubKeyType: "v0_p2wpkh",
	}

	recipientKey, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatal(err)
	}
	recipientHash := btcutil.Hash160(recipientKey.PubKey().SerializeCompressed())
	recipient, err := btcutil.NewAddressWitnessPubKeyHash(recipientHash, LitecoinTestNetParams)
	if err != nil {
		t.Fatal(err)
	}

	payload := BtcPayload{
		Recipient: recipient.EncodeAddress(),
		Amount:    500000,
		FeeRate:   10,
		Utxos:     []UTXO{utxo},
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}

	signedHex, err := a.CreateSignedTransaction(wallet, string(payloadJSON))
	if err != nil {
		t.Fatal(err)
	}

	signedBytes, err := hex.DecodeString(signedHex)
	if err != nil {
		t.Fatal(err)
	}
	var tx wire.MsgTx
	if err := tx.Deserialize(bytes.NewReader(signedBytes)); err != nil {
		t.Fatal(err)
	}

	vm, err := txscript.NewEngine(
		pkScript,
		&tx,
		0,
		txscript.StandardVerifyFlags,
		nil,
		nil,
		utxo.Value,
		txscript.NewCannedPrevOutputFetcher(pkScript, utxo.Value),
	)
	if err != nil {
		t.Fatalf("failed to create txscript.Engine: %v", err)
	}
	if err := vm.Execute(); err != nil {
		t.Errorf("transaction script execution failed: %v", err)
	}
}
