package adapters

import (
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/system"
)

func TestSolanaWalletDerivationAndImport(t *testing.T) {
	a := NewSolanaAdapter()

	wallet, err := a.DeriveWallet()
	if err != nil {
		t.Fatal(err)
	}

	privateKey, err := solana.PrivateKeyFromBase58(wallet.PrivateKey)
	if err != nil {
		t.Fatal(err)
	}
	if privateKey.PublicKey().String() != wallet.PublicKey {
		t.Fatalf("derived public key mismatch. got=%s want=%s", privateKey.PublicKey(), wallet.PublicKey)
	}

	imported, err := a.ImportWallet(wallet.PrivateKey)
	if err != nil {
		t.Fatal(err)
	}
	if imported.PublicKey != wallet.PublicKey {
		t.Fatalf("imported address mismatch. got=%s want=%s", imported.PublicKey, wallet.PublicKey)
	}

	keygenBytes := make([]int, len(privateKey))
	for i, value := range privateKey {
		keygenBytes[i] = int(value)
	}
	keygenJSON, err := json.Marshal(keygenBytes)
	if err != nil {
		t.Fatal(err)
	}
	importedFromKeygen, err := a.ImportWallet(string(keygenJSON))
	if err != nil {
		t.Fatal(err)
	}
	if importedFromKeygen.PublicKey != wallet.PublicKey {
		t.Fatalf("keygen import address mismatch. got=%s want=%s", importedFromKeygen.PublicKey, wallet.PublicKey)
	}
}

func TestSolanaCreateSignedTransactionBase64(t *testing.T) {
	a := NewSolanaAdapter()

	wallet, err := a.DeriveWallet()
	if err != nil {
		t.Fatal(err)
	}

	privateKey, err := solana.PrivateKeyFromBase58(wallet.PrivateKey)
	if err != nil {
		t.Fatal(err)
	}
	recipient, err := solana.NewRandomPrivateKey()
	if err != nil {
		t.Fatal(err)
	}

	tx, err := solana.NewTransaction(
		[]solana.Instruction{
			system.NewTransferInstruction(
				1_000,
				privateKey.PublicKey(),
				recipient.PublicKey(),
			).Build(),
		},
		solana.Hash{1, 2, 3},
		solana.TransactionPayer(privateKey.PublicKey()),
	)
	if err != nil {
		t.Fatal(err)
	}

	unsignedTx, err := tx.ToBase64()
	if err != nil {
		t.Fatal(err)
	}
	payloadJSON, err := json.Marshal(SolanaPayload{
		TransactionBase64: unsignedTx,
		OutputEncoding:    "base64",
	})
	if err != nil {
		t.Fatal(err)
	}

	signedBase64, err := a.CreateSignedTransaction(wallet, string(payloadJSON))
	if err != nil {
		t.Fatal(err)
	}

	signedTx, err := solana.TransactionFromBase64(signedBase64)
	if err != nil {
		t.Fatal(err)
	}
	if err := signedTx.VerifySignatures(); err != nil {
		t.Fatalf("signed transaction signature verification failed: %v", err)
	}
}

func TestSolanaCreateSignedTransactionHexOutput(t *testing.T) {
	a := NewSolanaAdapter()

	wallet, err := a.DeriveWallet()
	if err != nil {
		t.Fatal(err)
	}

	privateKey, err := solana.PrivateKeyFromBase58(wallet.PrivateKey)
	if err != nil {
		t.Fatal(err)
	}
	recipient, err := solana.NewRandomPrivateKey()
	if err != nil {
		t.Fatal(err)
	}

	tx, err := solana.NewTransaction(
		[]solana.Instruction{
			system.NewTransferInstruction(
				1_000,
				privateKey.PublicKey(),
				recipient.PublicKey(),
			).Build(),
		},
		solana.Hash{4, 5, 6},
		solana.TransactionPayer(privateKey.PublicKey()),
	)
	if err != nil {
		t.Fatal(err)
	}
	unsignedBytes, err := tx.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}

	payloadJSON, err := json.Marshal(SolanaPayload{
		TransactionHex: hex.EncodeToString(unsignedBytes),
		OutputEncoding: "hex",
	})
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
	signedTx, err := solana.TransactionFromBytes(signedBytes)
	if err != nil {
		t.Fatal(err)
	}
	if err := signedTx.VerifySignatures(); err != nil {
		t.Fatalf("signed transaction signature verification failed: %v", err)
	}
}
