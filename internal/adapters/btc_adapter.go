package adapters

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"

	btcec "github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
)

// TODO: use JSON playground to handle required feilds and types

type UTXO struct {
	Txid             string `json:"txid"`
	Value            int64  `json:"value"`
	ScriptPubKey     string `json:"script_pub_key"`
	ScriptPubKeyType string `json:"script_pubkey_type"`
	Vout             uint32 `json:"vout"`
}

type BtcPayload struct {
	Recipient string  `json:"recipient"`
	Amount    int64   `json:"amount"`
	FeeRate   float64 `json:"fee_rate"`
	Utxos     []UTXO  `json:"utxos"` // Details for each utxo
}

type btcAdapter struct {
	net *chaincfg.Params
}

func NewBtcAdapter(net *chaincfg.Params) *btcAdapter {
	return &btcAdapter{net: net}
}

func (a *btcAdapter) DeriveWallet() (*Wallet, error) {
	privateKey, err := btcec.NewPrivateKey()
	if err != nil {
		return nil, err
	}

	wif, err := btcutil.NewWIF(privateKey, a.net, true)
	if err != nil {
		return nil, err
	}

	addr, err := getPubKey(wif, a.net)
	if err != nil {
		return nil, fmt.Errorf("failed to get public key: %w", err)
	}

	return &Wallet{
		PrivateKey: wif.String(),
		PublicKey:  addr.EncodeAddress(),
	}, nil
}

func (a *btcAdapter) validatePayload(jsonPayload string) (*BtcPayload, error) {
	var payload BtcPayload
	if err := json.Unmarshal([]byte(jsonPayload), &payload); err != nil {
		return nil, fmt.Errorf("failed to decode payload: %w", err)
	}

	return &payload, nil
}

func (a *btcAdapter) CreateSignedTransaction(wallet *Wallet, payload string) (string, error) {
	btcPayload, err := a.validatePayload(payload)
	if err != nil {
		return "", fmt.Errorf("invalid payload type, expected BtcPayload: %w", err)
	}
	_ = btcPayload

	wif, err := btcutil.DecodeWIF(wallet.PrivateKey)
	if err != nil {
		return "", fmt.Errorf("failed to decode WIF: %w", err)
	}

	// Ensure the provided wallet belongs to the adapter's configured network.
	if !wif.IsForNet(a.net) {
		return "", fmt.Errorf("wif network mismatch: wallet WIF not for %s", a.net.Name)
	}

	tx, err := a.NewTxWithInputsAndOutputs(wif, btcPayload.Recipient, btcPayload.Amount, btcPayload.Utxos, btcPayload.FeeRate)
	if err != nil {

		return "", fmt.Errorf("failed to create transaction: %w", err)
	}
	var signedTx bytes.Buffer
	tx.Serialize(&signedTx)

	hexSignedTx := hex.EncodeToString(signedTx.Bytes())

	return hexSignedTx, nil
}

func (a *btcAdapter) NewTxWithInputsAndOutputs(wif *btcutil.WIF, destination string, amount int64, utxos []UTXO, feeRate float64) (*wire.MsgTx, error) {
	redeemTx := wire.NewMsgTx(wire.TxVersion)

	destinationAddr, err := btcutil.DecodeAddress(destination, a.net)
	if err != nil {
		return nil, err
	}

	// Ensure destination address is for the adapter's configured network
	if !destinationAddr.IsForNet(a.net) {
		return nil, fmt.Errorf("destination address not for %s", a.net.Name)
	}

	destinationAddrByte, err := txscript.PayToAddrScript(destinationAddr)
	if err != nil {
		return nil, err
	}

	// Derive change address (use the same P2WPKH address for simplicity)
	hash := btcutil.Hash160(wif.PrivKey.PubKey().SerializeCompressed())
	changeAddr, err := btcutil.NewAddressWitnessPubKeyHash(hash, a.net)
	// changeAddr, err := btcutil.NewAddressPubKey(wif.PrivKey.PubKey().SerializeUncompressed(), a.net)

	if err != nil {
		return nil, fmt.Errorf("failed to derive change address: %v", err)
	}

	if !changeAddr.IsForNet(a.net) {
		return nil, fmt.Errorf("change address not for %s", a.net.Name)
	}

	// README: THIS ALWAYS MIGRATES TO v0_p2wpkh ADDRESS. is this waht we want? what about specifiying the destination address
	changeAddrByte, err := txscript.PayToAddrScript(changeAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to create change script: %v", err)
	}

	var totalInputValue int64
	inputTypes := make([]string, 0, len(utxos))

	for _, utxo := range utxos {

		switch utxo.ScriptPubKeyType {
		case "v0_p2wpkh":
			expectedScript := append([]byte{0x00, 0x14}, hash...)
			if utxo.ScriptPubKey != hex.EncodeToString(expectedScript) {
				return nil, fmt.Errorf("UTXO scriptPubKey does not match wallet's P2WPKH address")
			}
		case "p2pkh":
			expectedScript := append(append([]byte{0x76, 0xa9, 0x14}, hash...), 0x88, 0xac)
			if utxo.ScriptPubKey != hex.EncodeToString(expectedScript) {
				return nil, fmt.Errorf("UTXO scriptPubKey does not match wallet's P2PKH address")
			}
		default:
			return nil, fmt.Errorf("unsupported script type: %s", utxo.ScriptPubKeyType)
		}

		utxoHash, err := chainhash.NewHashFromStr(utxo.Txid)
		if err != nil {
			return nil, err
		}

		outPoint := wire.NewOutPoint(utxoHash, utxo.Vout)

		// making the input, and adding it to transaction
		txIn := wire.NewTxIn(outPoint, nil, nil)
		redeemTx.AddTxIn(txIn)

		totalInputValue += utxo.Value
		inputTypes = append(inputTypes, utxo.ScriptPubKeyType)

	}

	// estimatedSize := len(utxos)*148 + 3*34 + 10 // inputs * 148 + outputs * 34 + overhead
	// estimatedFee := int64(estimatedSize) *
	destType, err := a.getOutputType(destination)
	if err != nil {
		return nil, fmt.Errorf("failed to determine destination type: %v", err)
	}

	feeInfo, err := CalculateFee(destType, inputTypes, amount, totalInputValue, feeRate)

	if err != nil {
		return nil, fmt.Errorf("failed to calculate fee: %v", err)
	}

	estimatedFee := feeInfo.EstimatedFee
	if amount <= 0 {
		return nil, fmt.Errorf("amount to send must be positive")
	}
	changeValue := totalInputValue - amount - estimatedFee
	if changeValue < 0 {
		return nil, fmt.Errorf("insufficient funds. Total: %d, Amount: %d, Fee: %d", totalInputValue, amount, estimatedFee)
	}

	txOut := wire.NewTxOut(int64(amount), destinationAddrByte)
	redeemTx.AddTxOut(txOut)

	// Add change output (if change is above dust threshold, e.g., 546 satoshis)
	if changeValue >= 546 {
		changeTxOut := wire.NewTxOut(int64(changeValue), changeAddrByte)
		redeemTx.AddTxOut(changeTxOut)
	} else if changeValue > 0 {
		// If change is below dust threshold, add it to the fee
		estimatedFee += int64(changeValue)
	}

	for idx, utxo := range utxos {

		witnessScript := utxo.ScriptPubKey

		sourcePKScript, err := hex.DecodeString(witnessScript)
		if err != nil {
			return nil, err
		}
		switch utxo.ScriptPubKeyType {
		case "v0_p2wpkh":
			sigHashes := txscript.NewTxSigHashes(redeemTx, txscript.NewCannedPrevOutputFetcher(
				sourcePKScript, utxo.Value))
			signature, err := txscript.WitnessSignature(redeemTx, sigHashes, idx, int64(utxo.Value), sourcePKScript, txscript.SigHashAll, wif.PrivKey, true)
			if err != nil {
				return nil, err
			}

			// Create witness stack
			redeemTx.TxIn[idx].Witness = signature
			redeemTx.TxIn[idx].SignatureScript = []byte{}
		case "p2pkh":
			signature, err := txscript.SignatureScript(redeemTx, idx, sourcePKScript,
				txscript.SigHashAll, wif.PrivKey, true)
			if err != nil {
				return nil, err
			}
			redeemTx.TxIn[idx].SignatureScript = signature
		default:
			return nil, fmt.Errorf("unsupported script type: %s", utxo.ScriptPubKeyType)

		}

	}

	return redeemTx, nil
}

func getPubKey(wif *btcutil.WIF, cfg *chaincfg.Params) (*btcutil.AddressWitnessPubKeyHash, error) {
	hash := btcutil.Hash160(wif.PrivKey.PubKey().SerializeCompressed())
	addr, err := btcutil.NewAddressWitnessPubKeyHash(hash, cfg)
	if err != nil {
		return nil, err
	}
	return addr, nil
}

func (a *btcAdapter) getOutputType(address string) (string, error) {
	addr, err := btcutil.DecodeAddress(address, a.net)
	if err != nil {
		return "", err
	}

	switch addr.(type) {
	case *btcutil.AddressWitnessPubKeyHash:
		return "p2wpkh", nil
	case *btcutil.AddressPubKeyHash:
		return "p2pkh", nil
	default:
		return "unknown", fmt.Errorf("unsupported address type")
	}
}
