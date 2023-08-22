package wallet

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math/rand"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/elnosh/btcw/tx"
)

var (
	ErrInsufficientFunds = errors.New("insufficient funds to make transaction")
)

func (w *Wallet) GetBalance() btcutil.Amount {
	return w.balance
}

func (w *Wallet) GetNewAddress() (string, error) {
	newKeyPair, err := w.generateNewExternalKeyPair()
	if err != nil {
		return "", err
	}

	return newKeyPair.Address, nil
}

func (w *Wallet) SendToAddress(address string, amount float64) (string, error) {
	amountToSend, err := btcutil.NewAmount(amount)
	if err != nil {
		return "", fmt.Errorf("invalid amount")
	}

	if w.balance < amountToSend {
		return "", ErrInsufficientFunds
	}

	// get estimate fee from btc node
	fee, err := w.client.EstimateFee(1)
	if err != nil {
		return "", fmt.Errorf("error estimating fee")
	}

	walletUtxos := make([]tx.UTXO, len(w.utxos))
	copy(walletUtxos, w.utxos)

	// need selected utxos to then use the derivation path
	// to get the private key to sign tx
	selectedUtxos := []tx.UTXO{}
	var currentAmount btcutil.Amount

	// add output to the tx with the script paying to address
	// and the amount
	txOut, err := tx.CreateTxOut(address, amountToSend)
	if err != nil {
		return "", err
	}
	txToSend := wire.NewMsgTx(wire.TxVersion)
	txToSend.AddTxOut(txOut)

	// build wire.MsgTx with randomly selected utxos as inputs
	for {
		// TODO: change to crypto rand
		// randomly select utxo
		max := len(w.utxos) - 1
		idx := rand.Intn(max)
		utxo := walletUtxos[idx]
		selectedUtxos = append(selectedUtxos, utxo)

		// delete utxo selected from copy (walletUtxos) so that if random generator
		// generates same number, same utxo won't be selected
		tx.DeleteUTXO(walletUtxos, idx)

		prevTxHash, err := chainhash.NewHashFromStr(utxo.TxID)
		if err != nil {
			return "", fmt.Errorf("error getting previous tx hash: %s", err.Error())
		}

		prevOut := wire.NewOutPoint(prevTxHash, utxo.VoutIdx)
		txIn := wire.NewTxIn(prevOut, nil, nil)
		txToSend.AddTxIn(txIn)

		currentAmount += utxo.Value

		// if already got value from utxos in wallet to satisfy amount
		// then calculate fee and change output before break
		if currentAmount > amountToSend {
			changeAmount := currentAmount - amountToSend
			//satsChangeAmount, _ := btcutil.NewAmount(changeAmount)

			newKeyPair, err := w.generateNewInternalKeyPair()
			if err != nil {
				return "", err
			}

			// add change output to transaction
			changeTxOut, err := tx.CreateTxOut(newKeyPair.Address, changeAmount)
			if err != nil {
				return "", err
			}
			txToSend.AddTxOut(changeTxOut)

			// calculate estimate total fee for the tx
			size := txToSend.SerializeSize()
			kbSize := float64(size) / float64(1000)
			estimateFee := fee * kbSize * 1.25

			feeAmount, _ := btcutil.NewAmount(estimateFee)

			// subtract fee from change
			txToSend.TxOut[len(txToSend.TxOut)-1].Value -= int64(feeAmount)
			break
		}
	}

	// at this point, txToSend must have all inputs and ouputs (including change output)
	// loop through the tx inputs, with derivation path for each selected utxo, get private key
	// and sign input at current idx with the private key
	for i, txin := range txToSend.TxIn {
		utxo := selectedUtxos[i]

		wif, err := w.getPrivateKeyForUTXO(utxo)
		if err != nil {
			return "", err
		}

		prevScript, err := hex.DecodeString(utxo.ScriptPubKey)
		if err != nil {
			return "", fmt.Errorf("error decoding scriptPubKey hex: %s", err.Error())
		}

		scripSig, err := txscript.SignatureScript(txToSend, i, prevScript, txscript.SigHashAll, wif.PrivKey, wif.CompressPubKey)
		if err != nil {
			return "", err
		}

		txin.SignatureScript = scripSig
	}

	for i := range txToSend.TxIn {
		utxo := selectedUtxos[i]
		prevScript, err := hex.DecodeString(utxo.ScriptPubKey)
		if err != nil {
			return "", fmt.Errorf("error decoding scriptPubKey hex: %s", err.Error())
		}

		vm, err := txscript.NewEngine(prevScript, txToSend, i, txscript.StandardVerifyFlags, nil, nil, int64(utxo.Value), nil)
		if err != nil {
			return "", err
		}

		err = vm.Execute()
		if err != nil {
			return "", fmt.Errorf("error validating tx: %s", err.Error())
		}
	}

	_, err = w.client.SendRawTransaction(txToSend, true)
	if err != nil {
		return "", fmt.Errorf("error sending transaction: %s", err.Error())
	}

	return txToSend.TxHash().String(), nil
}
