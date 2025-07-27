package adapters

import (
	"fmt"

	"github.com/shopspring/decimal"
)

func calculateTransactionSize(numP2PKHInputs, numP2WPKHInputs, numP2PKHOutputs, numP2WPKHOutputs int) int {
	inputSize := numP2PKHInputs*148 + numP2WPKHInputs*68
	outputSize := numP2PKHOutputs*34 + numP2WPKHOutputs*31
	overhead := 10

	// Add witness overhead if there are segwit inputs
	if numP2WPKHInputs > 0 || numP2WPKHOutputs > 0 {
		overhead += 1 // witness marker/flag amortized
	}

	return inputSize + outputSize + overhead
}

func countInputsByType(inputTypes []string) (p2pkh, p2wpkh int) {
	for _, inputType := range inputTypes {
		switch inputType {
		case "p2pkh":
			p2pkh++
		case "v0_p2wpkh":
			p2wpkh++
		}
	}
	return p2pkh, p2wpkh
}

type FeeInfo struct {
	EstimatedFee int64
	ChangeValue  int64
	NumOutputs   int
	TxSize       int
}

/*
README: calculate FEE ASSUMES CHAIN ADDRESS IS v0_p2wpkh
so if there is change the numP2WPKHOutputs increases by 1 thus affecting the gas cost
*/
func CalculateFee(destType string, inputTypes []string, amount, totalInputValue int64, feeRate float64) (*FeeInfo, error) {
	const dustThreshold = 546

	// Count inputs
	numP2PKHInputs, numP2WPKHInputs := countInputsByType(inputTypes)

	// Set destination output
	numP2PKHOutputs := 0
	numP2WPKHOutputs := 1
	if destType == "p2pkh" {
		numP2PKHOutputs = 1
		numP2WPKHOutputs = 0
	}

	// Step 1: Calculate transaction size without change
	txSizeNoChange := calculateTransactionSize(numP2PKHInputs, numP2WPKHInputs, numP2PKHOutputs, numP2WPKHOutputs)
	estimatedFeeNoChange := decimal.NewFromFloat(feeRate).Mul(decimal.NewFromInt(int64(txSizeNoChange))).Round(0).IntPart()

	// Check for insufficient funds
	if totalInputValue < amount+estimatedFeeNoChange {
		return nil, fmt.Errorf("insufficient funds. Total: %d, Amount: %d, Fee: %d", totalInputValue, amount, estimatedFeeNoChange)
	}

	// Calculate potential change without change output
	changeValue := totalInputValue - amount - estimatedFeeNoChange

	// Step 2: If change is exactly 0, return no-change result
	if changeValue == 0 {
		return &FeeInfo{
			EstimatedFee: estimatedFeeNoChange,
			ChangeValue:  0,
			NumOutputs:   1, // Only destination
			TxSize:       txSizeNoChange,
		}, nil
	}

	// Step 3: If change is dust, add to fee and return no-change result
	if changeValue > 0 && changeValue < dustThreshold {
		finalFee := estimatedFeeNoChange + changeValue
		return &FeeInfo{
			EstimatedFee: finalFee,
			ChangeValue:  0,
			NumOutputs:   1, // Only destination
			TxSize:       txSizeNoChange,
		}, nil
	}

	// Step 4: Try with change output if change is sufficient
	if changeValue >= dustThreshold {
		txSizeWithChange := calculateTransactionSize(numP2PKHInputs, numP2WPKHInputs, numP2PKHOutputs, numP2WPKHOutputs+1)
		estimatedFeeWithChange := decimal.NewFromFloat(feeRate).Mul(decimal.NewFromInt(int64(txSizeWithChange))).Round(0).IntPart()
		// Check if funds are sufficient with change
		if totalInputValue >= amount+estimatedFeeWithChange {
			newChangeValue := totalInputValue - amount - estimatedFeeWithChange
			if newChangeValue >= dustThreshold {
				return &FeeInfo{
					EstimatedFee: estimatedFeeWithChange,
					ChangeValue:  newChangeValue,
					NumOutputs:   2, // Destination + change
					TxSize:       txSizeWithChange,
				}, nil
			}
		}
	}

	// Step 5: No change output (insufficient funds for change or dust)
	finalFee := estimatedFeeNoChange
	if changeValue > 0 {
		finalFee += changeValue
		changeValue = 0
	}

	return &FeeInfo{
		EstimatedFee: finalFee,
		ChangeValue:  changeValue,
		NumOutputs:   1, // Only destination
		TxSize:       txSizeNoChange,
	}, nil
}
