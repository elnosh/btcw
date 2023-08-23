package tx

import (
	"errors"
	"math/rand"

	"github.com/btcsuite/btcd/btcutil"
)

type UTXO struct {
	TxID           string
	VoutIdx        uint32
	Value          btcutil.Amount
	ScriptPubKey   string
	Spent          bool
	DerivationPath string // path of the key associated with this utxo
}

func NewUTXO(txid string, voutIdx uint32, value btcutil.Amount, script, path string) *UTXO {
	return &UTXO{TxID: txid, VoutIdx: voutIdx, Value: value, ScriptPubKey: script,
		Spent: false, DerivationPath: path}
}

// deletes a utxo from the slice
func DeleteUTXO(utxos []UTXO, idx int) {
	utxos[idx] = utxos[len(utxos)-1]
	utxos = utxos[:len(utxos)-1]
}

// SelectUTXOs will take a desired amount to send and a list of utxos
// to select from to fulfill that amount. It will return a list of randomly
// selected utxos and the amount those utxos add up to. If
// there is not enough to fulfill the amount, it will return an error
func SelectUTXOs(amountToSend btcutil.Amount, utxos []UTXO) ([]UTXO, btcutil.Amount, error) {
	// copy utxos slice to then delete from copy
	// without affecting original utxos slice passed
	utxosCopy := make([]UTXO, len(utxos))
	copy(utxosCopy, utxos)

	if len(utxos) == 0 {
		return nil, 0, errors.New("no utxos to select")
	} else if len(utxos) == 1 {
		utxosAmount := utxos[0].Value
		if utxosAmount > amountToSend {
			return utxosCopy, utxosAmount, nil
		} else {
			return nil, 0, ErrInsufficientAmount
		}
	}

	selectedUtxos := []UTXO{}
	var selectedAmount btcutil.Amount

	for {
		// TODO: change to crypto rand
		max := len(utxosCopy) - 1
		idx := rand.Intn(max)

		utxo := utxosCopy[idx]
		selectedUtxos = append(selectedUtxos, utxo)
		// delete selected utxo so that it does not
		// get selected again
		DeleteUTXO(utxosCopy, idx)

		selectedAmount += utxo.Value

		if selectedAmount > amountToSend {
			break
		} else {
			// if there is only 1 utxo left and amountToSend has not been
			// reached yet, check if with last utxo value will be enough.
			// if it is, add it up and if not then return err
			if len(utxosCopy) == 1 {
				if selectedAmount+utxosCopy[0].Value > amountToSend {
					selectedAmount += utxosCopy[0].Value
					selectedUtxos = append(selectedUtxos, utxosCopy[0])
					break
				} else {
					return nil, 0, ErrInsufficientAmount
				}
			}
		}
	}

	return selectedUtxos, selectedAmount, nil
}
