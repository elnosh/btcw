package wallet

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math/rand"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
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
	satsAmount, err := btcutil.NewAmount(amount)
	if err != nil {
		return "", fmt.Errorf("invalid amount")
	}

	if w.balance < satsAmount {
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
	var currentAmount float64

	// build tx output with the script and the amount (wanting to be sent)
	addr, err := btcutil.DecodeAddress(address, &chaincfg.SimNetParams)
	if err != nil {
		return "", fmt.Errorf("invalid address")
	}
	script, err := txscript.PayToAddrScript(addr)
	if err != nil {
		return "", err
	}
	tx := wire.NewMsgTx(wire.TxVersion)
	txOut := wire.NewTxOut(int64(satsAmount), script)
	tx.AddTxOut(txOut)

	// build wire.MsgTx with randomly selected utxos as inputs
	for {
		max := len(w.utxos) - 1

		// TODO: change to crypto rand
		idx := rand.Intn(max)
		utxo := walletUtxos[idx]
		selectedUtxos = append(selectedUtxos, utxo)

		// delete utxo selected from copy (walletUtxos) so that if random generator
		// generates same number, same utxo won't be selected
		walletUtxos[idx] = walletUtxos[len(walletUtxos)-1]
		walletUtxos = walletUtxos[:len(walletUtxos)-1]

		prevTxHash, err := chainhash.NewHashFromStr(utxo.TxID)
		if err != nil {
			return "", fmt.Errorf("error getting previous tx hash: %s", err.Error())
		}

		prevOut := wire.NewOutPoint(prevTxHash, utxo.VoutIdx)
		txIn := wire.NewTxIn(prevOut, nil, nil)
		tx.AddTxIn(txIn)

		currentAmount += utxo.Value

		// if already got value from utxos in wallet to satisfy amount
		// then calculate fee and change output before break
		if currentAmount > amount {
			changeAmount := currentAmount - amount

			newKeyPair, err := w.generateNewInternalKeyPair()
			if err != nil {
				return "", err
			}

			changeAddr, err := btcutil.DecodeAddress(newKeyPair.Address, &chaincfg.SimNetParams)
			if err != nil {
				return "", fmt.Errorf("error decoding address: %s", err.Error())
			}
			changeOutputScript, err := txscript.PayToAddrScript(changeAddr)
			if err != nil {
				return "", err
			}

			satsChangeAmount, err := btcutil.NewAmount(changeAmount)
			if err != nil {
				return "", err
			}

			changeTxOut := wire.NewTxOut(int64(satsChangeAmount), changeOutputScript)
			tx.AddTxOut(changeTxOut)

			// calculate estimate of total fee for the tx
			size := tx.SerializeSize()
			// or float64(size) / float64(1000)
			kbSize := float64(size / 1000)
			estimateFee := fee * kbSize * 1.25

			feeAmount, err := btcutil.NewAmount(estimateFee)
			if err != nil {
				return "", err
			}

			// need another check after adding fee and that there are enough funds in wallet
			// subtract fee from change
			tx.TxOut[len(tx.TxOut)-1].Value -= int64(feeAmount)
			break
		}
	}

	// at this point, msg.tx must have all inputs and ouputs (including change output)
	// loop through the tx inputs, with derivation path for each selected utxo, get private key
	// and sign input at current idx with the private key
	for i, txin := range tx.TxIn {
		derivationPath := selectedUtxos[i].DerivationPath
		kp := w.getKeyPair(derivationPath)
		if kp == nil {
			return "", fmt.Errorf("error signing transaction")
		}

		passKey, err := w.GetDecodedKey()
		if err != nil {
			return "", err
		}

		wifStr, err := Decrypt(kp.EncryptedPrivateKey, passKey)
		if err != nil {
			return "", err
		}

		wif, err := btcutil.DecodeWIF(string(wifStr))
		if err != nil {
			return "", fmt.Errorf("error decoding wif: %s", err.Error())
		}

		prevScript, err := hex.DecodeString(selectedUtxos[i].ScriptPubKey)
		if err != nil {
			return "", fmt.Errorf("error decoding scriptPubKey hex: %s", err.Error())
		}

		scripSig, err := txscript.SignatureScript(tx, i, prevScript, txscript.SigHashAll, wif.PrivKey, wif.CompressPubKey)
		if err != nil {
			return "", err
		}

		txin.SignatureScript = scripSig
	}

	for i := range tx.TxIn {
		utxo := selectedUtxos[i]
		prevScript, err := hex.DecodeString(utxo.ScriptPubKey)
		if err != nil {
			return "", fmt.Errorf("error decoding scriptPubKey hex: %s", err.Error())
		}
		amount, err := btcutil.NewAmount(utxo.Value)
		if err != nil {
			return "", err
		}

		vm, err := txscript.NewEngine(prevScript, tx, i, txscript.StandardVerifyFlags, nil, nil, int64(amount), nil)
		if err != nil {
			return "", err
		}

		err = vm.Execute()
		if err != nil {
			return "", fmt.Errorf("error validating tx: %s", err.Error())
		}
	}

	_, err = w.client.SendRawTransaction(tx, true)
	if err != nil {
		return "", fmt.Errorf("error sending transaction: %s", err.Error())
	}

	return tx.TxHash().String(), nil
}
