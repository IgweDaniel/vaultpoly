package adapters

import (
	"testing"
)

// Test case structure
type TestCase struct {
	Name            string
	TotalInput      int64
	Amount          int64
	FeeRate         float64
	InputTypes      []string // "p2pkh" or "v0_p2wpkh"
	DestinationType string   // "p2pkh" or "p2wpkh"

	// Expected results
	ExpectedFee     int64
	ExpectedChange  int64
	ExpectedOutputs int  // 1 = destination only, 2 = destination + change
	ExpectedTxSize  int  // in vbytes
	ExpectError     bool // true if we expect an error (e.g., insufficient funds)
}

var testCases = []TestCase{
	{
		Name:            "Basic P2PKH to P2PKH with change",
		TotalInput:      100000,
		Amount:          50000,
		FeeRate:         1,
		InputTypes:      []string{"p2pkh"},
		DestinationType: "p2pkh",

		ExpectedFee:     224,   // (148 + 34 + 31 + 11) * 1 = 224 (P2PKH input + P2PKH dest + P2WPKH change + segwit overhead)
		ExpectedChange:  49776, // 100000 - 50000 - 224 = 49776
		ExpectedOutputs: 2,
		ExpectedTxSize:  224,
	},

	{
		Name:            "P2PKH to P2WPKH with change",
		TotalInput:      100000,
		Amount:          50000,
		FeeRate:         1,
		InputTypes:      []string{"p2pkh"},
		DestinationType: "p2wpkh",

		ExpectedFee:     221,   // (148 + 31 + 31 + 11) * 1 = 221
		ExpectedChange:  49779, // 100000 - 50000 - 221 = 49779
		ExpectedOutputs: 2,
		ExpectedTxSize:  221,
	},

	{
		Name:            "P2WPKH to P2WPKH with change",
		TotalInput:      100000,
		Amount:          50000,
		FeeRate:         1,
		InputTypes:      []string{"v0_p2wpkh"},
		DestinationType: "p2wpkh",

		ExpectedFee:     141,   // (68 + 31 + 31 + 11) * 1 = 141
		ExpectedChange:  49859, // 100000 - 50000 - 141 = 49859
		ExpectedOutputs: 2,
		ExpectedTxSize:  141,
	},

	{
		Name:            "Multiple P2PKH inputs",
		TotalInput:      100000,
		Amount:          50000,
		FeeRate:         1,
		InputTypes:      []string{"p2pkh", "p2pkh"},
		DestinationType: "p2wpkh",

		ExpectedFee:     369,   // (148*2 + 31 + 31 + 11) * 1 = 369
		ExpectedChange:  49631, // 100000 - 50000 - 369 = 49631
		ExpectedOutputs: 2,
		ExpectedTxSize:  369,
	},

	{
		Name:            "Mixed inputs (P2PKH + P2WPKH)",
		TotalInput:      100000,
		Amount:          50000,
		FeeRate:         1,
		InputTypes:      []string{"p2pkh", "v0_p2wpkh"},
		DestinationType: "p2wpkh",

		ExpectedFee:     289,   // (148 + 68 + 31 + 31 + 11) * 1 = 289
		ExpectedChange:  49711, // 100000 - 50000 - 289 = 49711
		ExpectedOutputs: 2,
		ExpectedTxSize:  289,
	},

	{
		Name:            "Change exactly at dust threshold",
		TotalInput:      50767, // Calculated to make change exactly 546
		Amount:          50000,
		FeeRate:         1,
		InputTypes:      []string{"p2pkh"},
		DestinationType: "p2wpkh",

		ExpectedFee:     221, // (148 + 31 + 31 + 11) * 1 = 221
		ExpectedChange:  546, // 50767 - 50000 - 221 = 546 (exactly dust threshold)
		ExpectedOutputs: 2,
		ExpectedTxSize:  221,
	},

	{
		Name:            "Change just below dust threshold - becomes fee",
		TotalInput:      50766, // One sat less than above
		Amount:          50000,
		FeeRate:         1,
		InputTypes:      []string{"p2pkh"},
		DestinationType: "p2wpkh",

		ExpectedFee:     766, // Base: (148 + 31 + 11) * 1 = 190, plus dust: 50766 - 50000 - 221 = 545, total = 190 + 545 = 735 (but since 545 < 546, no change output, fee = 766)
		ExpectedChange:  0,   // No change output
		ExpectedOutputs: 1,
		ExpectedTxSize:  190, // Without change output
	},

	{
		Name:            "Large dust amount added to fee",
		TotalInput:      50500,
		Amount:          50000,
		FeeRate:         1,
		InputTypes:      []string{"p2pkh"},
		DestinationType: "p2wpkh",

		ExpectedFee:     500, // Base: (148 + 31 + 11) * 1 = 190, dust: 50500 - 50000 - 190 = 310, total = 500
		ExpectedChange:  0,
		ExpectedOutputs: 1,
		ExpectedTxSize:  190,
	},

	{
		Name:            "Exact amount - no change",
		TotalInput:      50190, // Exactly amount + fee
		Amount:          50000,
		FeeRate:         1,
		InputTypes:      []string{"p2pkh"},
		DestinationType: "p2wpkh",

		ExpectedFee:     190, // (148 + 31 + 11) * 1 = 190
		ExpectedChange:  0,
		ExpectedOutputs: 1,
		ExpectedTxSize:  190,
	},

	{
		Name:            "High fee rate with change",
		TotalInput:      100000,
		Amount:          50000,
		FeeRate:         10, // Higher fee rate
		InputTypes:      []string{"p2pkh"},
		DestinationType: "p2wpkh",

		ExpectedFee:     2210,  // (148 + 31 + 31 + 11) * 10 = 2210
		ExpectedChange:  47790, // 100000 - 50000 - 2210 = 47790
		ExpectedOutputs: 2,
		ExpectedTxSize:  221,
	},

	{
		Name:            "P2WPKH dust threshold test",
		TotalInput:      50687, // Calculated for P2WPKH
		Amount:          50000,
		FeeRate:         1,
		InputTypes:      []string{"v0_p2wpkh"},
		DestinationType: "p2wpkh",

		ExpectedFee:     141, // (68 + 31 + 31 + 11) * 1 = 141
		ExpectedChange:  546, // 50687 - 50000 - 141 = 546
		ExpectedOutputs: 2,
		ExpectedTxSize:  141,
	},

	{
		Name:            "P2WPKH just below dust",
		TotalInput:      50686, // One less
		Amount:          50000,
		FeeRate:         1,
		InputTypes:      []string{"v0_p2wpkh"},
		DestinationType: "p2wpkh",

		ExpectedFee:     686, // Base: (68 + 31 + 11) * 1 = 110, dust: 50686 - 50000 - 141 = 545, total = 110 + 545 = 655 (but since 545 < 546, no change output, fee = 686)
		ExpectedChange:  0,
		ExpectedOutputs: 1,
		ExpectedTxSize:  110,
	},

	{
		Name:            "Insufficient funds",
		TotalInput:      1000,
		Amount:          50000,
		FeeRate:         1,
		InputTypes:      []string{"p2pkh"},
		DestinationType: "p2wpkh",

		ExpectedFee:     -1, // Special value indicating insufficient funds
		ExpectedChange:  -1,
		ExpectedOutputs: -1,
		ExpectedTxSize:  -1,
		ExpectError:     true, // Expect an error due to insufficient funds
	},
}

func TestFeeCalculationCases(t *testing.T) {
	for i, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {

			feeInfo, err := CalculateFee(tc.DestinationType, tc.InputTypes, tc.Amount, tc.TotalInput, tc.FeeRate)
			if err != nil && !tc.ExpectError {
				t.Errorf("Test %d (%s): unexpected error: %v", i+1, tc.Name, err)
				return
			}
			if err == nil && tc.ExpectError {
				t.Errorf("Test %d (%s): expected error but got none", i+1, tc.Name)
				return
			}

			if tc.ExpectError {
				// Special case: insufficient funds
				if feeInfo != nil {
					t.Errorf("Test %d (%s): expected error for insufficient funds, got feeInfo: %v", i+1, tc.Name, feeInfo)
				}
				return
			}
			// Replace the following lines with actual calculation logic.
			fee := feeInfo.EstimatedFee
			change := feeInfo.ChangeValue
			outputs := feeInfo.NumOutputs
			size := feeInfo.TxSize

			if fee != tc.ExpectedFee {
				t.Errorf("Test %d (%s): expected fee %d, got %d", i+1, tc.Name, tc.ExpectedFee, fee)
			}
			if change != tc.ExpectedChange {
				t.Errorf("Test %d (%s): expected change %d, got %d", i+1, tc.Name, tc.ExpectedChange, change)
			}
			if outputs != tc.ExpectedOutputs {
				t.Errorf("Test %d (%s): expected outputs %d, got %d", i+1, tc.Name, tc.ExpectedOutputs, outputs)
			}
			if size != tc.ExpectedTxSize {
				t.Errorf("Test %d (%s): expected tx size %d, got %d", i+1, tc.Name, tc.ExpectedTxSize, size)
			}
		})
	}
}
