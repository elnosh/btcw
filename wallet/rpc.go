package wallet

import (
	"errors"
	"fmt"
	"time"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/elnosh/btcw/tx"
	"github.com/elnosh/btcw/utils"
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
	if w.locked {
		return "", fmt.Errorf("wallet is locked. unlock wallet with 'walletpassphrase' command first.")
	}

	amountToSend, err := btcutil.NewAmount(amount)
	if err != nil {
		return "", fmt.Errorf("invalid amount")
	}

	if w.balance < amountToSend {
		return "", ErrInsufficientFunds
	}

	// get estimate fee from btc node
	fee := w.client.EstimateFee(1)

	// select utxos from wallet to fulfill amountToSend
	selectedUtxos, _, err := tx.SelectUTXOs(amountToSend, w.utxos)
	if err != nil {
		w.LogError("unable to send to address - error selecting UTXOs: %s", err.Error())
		return "", err
	}

	// create unsigned tx from the selected utxos
	txToSend, err := w.createRawTransaction(address, amountToSend, fee, selectedUtxos)
	if err != nil {
		w.LogError("unable to send to address - error creating transaction: %s", err.Error())
		return "", err
	}

	// sign raw transaction using keys associated with selected utxos
	err = w.signTransaction(txToSend, selectedUtxos)
	if err != nil {
		w.LogError("unable to send to address - error signing transaction: %s", err.Error())
		return "", fmt.Errorf("error signing transaction: %s", err.Error())
	}

	err = validateSignedTransaction(txToSend, selectedUtxos)
	if err != nil {
		w.LogError("unable to send to address - invalid transaction: %s", err.Error())
		return "", fmt.Errorf("error validating transaction: %s", err.Error())
	}

	// send tx to network after validating
	_, err = w.client.SendRawTransaction(txToSend, true)
	if err != nil {
		w.LogError("error sending transaction to network: %s", err.Error())
		return "", fmt.Errorf("error sending transaction: %s", err.Error())
	}
	w.LogInfo("sent tx %s to network", txToSend.TxHash().String())

	// if no errors while creating and broadcasting tx, then update the wallet fields
	// accordingly. Subtract amount sent and fee from balance, mark spent utxos
	// and add change utxo to list of wallet utxos
	w.updateWalletAfterTx(txToSend, selectedUtxos, amountToSend)

	return txToSend.TxHash().String(), nil
}

func (w *Wallet) WalletPassphrase(passphrase string, duration time.Duration) error {
	encodedHash := string(w.getEncodedHash())

	if !utils.VerifyPassphrase(encodedHash, passphrase) {
		return errors.New("invalid passphrase")
	}

	w.unlock(duration)
	return nil
}
