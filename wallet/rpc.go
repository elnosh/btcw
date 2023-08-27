package wallet

import (
	"errors"
	"fmt"

	"github.com/btcsuite/btcd/btcutil"
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
	btcfee := w.client.EstimateFee(1)
	fee, err := btcutil.NewAmount(btcfee)
	if err != nil {
		return "", fmt.Errorf("error estimating fee")
	}

	// select utxos from wallet to fulfill amountToSend
	selectedUtxos, _, err := tx.SelectUTXOs(amountToSend, w.utxos)
	if err != nil {
		return "", err
	}

	// create unsigned tx from the selected utxos
	txToSend, err := w.createRawTransaction(address, amountToSend, fee, selectedUtxos)
	if err != nil {
		return "", err
	}

	// sign raw transaction using keys associated with selected utxos
	err = w.signTransaction(txToSend, selectedUtxos)
	if err != nil {
		return "", fmt.Errorf("error signing transaction: %s", err.Error())
	}

	err = validateSignedTransaction(txToSend, selectedUtxos)
	if err != nil {
		return "", fmt.Errorf("error validating transaction: %s", err.Error())
	}

	// send tx to network after validating
	_, err = w.client.SendRawTransaction(txToSend, true)
	if err != nil {
		return "", fmt.Errorf("error sending transaction: %s", err.Error())
	}

	// if no errors while creating and broadcasting tx, then update the wallet fields
	// accordingly. Subtract amount sent and fee from balance, mark spent utxos
	// and add change utxo to list of wallet utxos
	w.updateWalletAfterTx(txToSend, selectedUtxos, amountToSend)

	return txToSend.TxHash().String(), nil
}
