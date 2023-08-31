package tx

import (
	"testing"

	"github.com/btcsuite/btcd/btcutil"
)

func TestSelectUTXOs(t *testing.T) {
	tests := []struct {
		name             string
		amountToSend     btcutil.Amount
		utxos            []UTXO
		wantedAmount     btcutil.Amount
		numSelectedUTXOs int
		wantedErr        error
	}{
		{
			name:         "one utxo meet amount to send",
			amountToSend: btcutil.Amount(30000),
			utxos: []UTXO{
				*NewUTXO("txid1", 1, 70000, nil, ""),
			},
			wantedAmount:     btcutil.Amount(70000),
			numSelectedUTXOs: 1,
			wantedErr:        nil,
		},
		{
			name:         "UTXOs meet amount to send",
			amountToSend: btcutil.Amount(80000),
			utxos: []UTXO{
				*NewUTXO("txid1", 1, 30000, nil, ""),
				*NewUTXO("txid1", 6, 60000, nil, ""),
			},
			wantedAmount:     btcutil.Amount(90000),
			numSelectedUTXOs: 2,
			wantedErr:        nil,
		},
		{
			name:         "insufficient amount one utxo",
			amountToSend: btcutil.Amount(80000),
			utxos: []UTXO{
				*NewUTXO("txid1", 1, 30000, nil, ""),
			},
			wantedAmount:     btcutil.Amount(0),
			numSelectedUTXOs: 0,
			wantedErr:        ErrInsufficientAmount,
		},
		{
			name:         "insufficient amount",
			amountToSend: btcutil.Amount(140000),
			utxos: []UTXO{
				*NewUTXO("txid1", 1, 50000, nil, ""),
				*NewUTXO("txid2", 1, 10000, nil, ""),
				*NewUTXO("txid2", 2, 30000, nil, ""),
			},
			wantedAmount:     btcutil.Amount(0),
			numSelectedUTXOs: 0,
			wantedErr:        ErrInsufficientAmount,
		},
		{
			name:             "no utxos",
			amountToSend:     btcutil.Amount(40000),
			utxos:            []UTXO{},
			wantedAmount:     btcutil.Amount(0),
			numSelectedUTXOs: 0,
			wantedErr:        ErrNoUTXOs,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			selectedUTXOs, selectedAmount, err := SelectUTXOs(test.amountToSend, test.utxos)
			selectedLen := len(selectedUTXOs)

			if selectedLen != test.numSelectedUTXOs {
				t.Errorf("number of selected UTXOs do not match - expected: %v, got: %v", test.numSelectedUTXOs, selectedLen)
			}

			if selectedAmount != test.wantedAmount {
				t.Errorf("selected amount does match - expected: %v, got: %v", test.wantedAmount, selectedAmount)
			}

			if err != test.wantedErr {
				t.Errorf("errors do not match - expected: %v, got %v", test.wantedErr, err)
			}
		})
	}

	t.Run("multiple UTXOs", func(t *testing.T) {
		utxo1 := NewUTXO("txid1", 1, 30000, nil, "")
		utxo2 := NewUTXO("txid2", 0, 10000, nil, "")
		utxo3 := NewUTXO("txid1", 8, 70000, nil, "")
		utxo4 := NewUTXO("txid3", 1, 110000, nil, "")
		utxo5 := NewUTXO("txid438", 2, 10000, nil, "")
		utxo6 := NewUTXO("txid11", 1, 80000, nil, "")
		utxo7 := NewUTXO("txid11", 2, 990000, nil, "")
		utxos := []UTXO{*utxo1, *utxo2, *utxo3, *utxo4, *utxo5, *utxo6, *utxo7}

		amountToSend := btcutil.Amount(125000)

		selectedUTXOs, selectedAmount, err := SelectUTXOs(amountToSend, utxos)
		for _, selected := range selectedUTXOs {
			if selected.Spent {
				t.Errorf("error selected spent UTXO - %v", selected)
			}
		}

		if selectedAmount < amountToSend {
			t.Errorf("did not select sufficient UTXOs - amount: %v", selectedAmount)
		}

		if err != nil {
			t.Errorf("expected: %v error, got %v", nil, err)
		}
	})
}
