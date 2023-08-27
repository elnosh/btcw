package wallet

import (
	"fmt"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/elnosh/btcw/tx"
)

// createRawTransaction will create an unsigned tx to the address
// and amountToSend. It will create it from the set of utxos passed and
// it will add change output if needed
func (w *Wallet) createRawTransaction(address string, amountToSend, feeRate btcutil.Amount, utxos []tx.UTXO) (*wire.MsgTx, error) {
	txOut, err := tx.CreateTxOut(address, amountToSend, w.network)
	if err != nil {
		return nil, err
	}
	rawTx := wire.NewMsgTx(wire.TxVersion)
	rawTx.AddTxOut(txOut)

	totalUtxosAmount := btcutil.Amount(0)
	for _, utxo := range utxos {
		txIn, err := tx.CreateTxIn(utxo)
		if err != nil {
			return nil, err
		}
		rawTx.AddTxIn(txIn)

		totalUtxosAmount += utxo.Value
	}

	if totalUtxosAmount > amountToSend {
		changeAmount := totalUtxosAmount - amountToSend

		// get new internal key for change
		newInternalKey, err := w.generateNewInternalKeyPair()
		if err != nil {
			return nil, err
		}

		// create change output from new internal address and change amount
		changeTxOut, err := tx.CreateTxOut(newInternalKey.Address, changeAmount, w.network)
		if err != nil {
			return nil, err
		}
		rawTx.AddTxOut(changeTxOut)

		// calculate estimate fee for tx
		size := rawTx.SerializeSize()
		kbSize := int64(size) / 1000
		estimateFee := kbSize * int64(feeRate) * 2

		// subtract fee from change
		rawTx.TxOut[len(rawTx.TxOut)-1].Value -= estimateFee
	} else {
		return nil, tx.ErrInsufficientAmount
	}

	return rawTx, nil
}

// signTransaction will sign all inputs in tx using the keys associated with
// the utxos referenced
func (w *Wallet) signTransaction(tx *wire.MsgTx, utxos []tx.UTXO) error {
	for i, txIn := range tx.TxIn {
		utxo := utxos[i]

		// get private key that can create valid signature to spend utxo
		wif, err := w.getPrivateKeyForUTXO(utxo)
		if err != nil {
			return err
		}

		scriptSig, err := txscript.SignatureScript(tx, i, utxo.ScriptPubKey,
			txscript.SigHashAll, wif.PrivKey, wif.CompressPubKey)
		if err != nil {
			return err
		}
		txIn.SignatureScript = scriptSig
	}

	return nil
}

func validateSignedTransaction(tx *wire.MsgTx, utxos []tx.UTXO) error {
	for i := range tx.TxIn {
		utxo := utxos[i]

		vm, err := txscript.NewEngine(utxo.ScriptPubKey, tx, i, txscript.StandardVerifyFlags, nil, nil, int64(utxo.Value), nil)
		if err != nil {
			return err
		}

		err = vm.Execute()
		if err != nil {
			return err
		}
	}

	return nil
}

// extractTxInfo takes in a wire tx, utxos used as input and amount sent. From that,
// it will return the change output, index of change output and fee
func extractTxInfo(txMsg *wire.MsgTx, usedUTXOs []tx.UTXO, amountToSend btcutil.Amount) (wire.TxOut, uint32, btcutil.Amount) {
	var totalOutput int64 = 0
	var totalInput int64 = 0

	for _, utxo := range usedUTXOs {
		totalInput += int64(utxo.Value)
	}

	var changeOutput wire.TxOut
	changeIdx := 0
	for idx, txout := range txMsg.TxOut {
		totalOutput += txout.Value

		if txout.Value != int64(amountToSend) {
			changeOutput = *txout
			changeIdx = idx
		}
	}
	fee := btcutil.Amount(totalInput - totalOutput)

	return changeOutput, uint32(changeIdx), fee
}

// updateWalletAfterTx will update wallet fields based on the transaction sent.
// It will mark utxos used in the tx as spent, add change UTXO to wallet and update
// the balance
func (w *Wallet) updateWalletAfterTx(txMsg *wire.MsgTx, usedUTXOs []tx.UTXO, amountToSend btcutil.Amount) {
	changeOutput, changeIdx, fee := extractTxInfo(txMsg, usedUTXOs, amountToSend)

	// mark utxos used to create transaction as spent
	go w.markSpentUTXOs(usedUTXOs)

	// add change utxo to wallet utxo list
	go w.addChangeUTXO(txMsg, changeOutput, changeIdx)

	// new balance will be current wallet balance - amount wanting to be sent - fee
	newBalance := w.balance - amountToSend - fee
	go w.setBalance(newBalance)
}

// markSpentUTXOs takes a list of utxos and if it finds them in the wallet
// it will mark them as spent
func (w *Wallet) markSpentUTXOs(utxos []tx.UTXO) {
	for _, utxo := range utxos {
		for _, walletUtxo := range w.utxos {
			if walletUtxo.TxID == utxo.TxID {
				// update in db and struct
				utxo.Spent = true
				key := utxo.GetOutpoint()
				err := w.updateUTXO(key, utxo)
				// only update utxo in wallet struct if update in db succeeded
				if err == nil {
					walletUtxo.Spent = true
				}
				break
			}
		}
	}
}

// adds change output to wallet utxos
func (w *Wallet) addChangeUTXO(txMsg *wire.MsgTx, changeOutput wire.TxOut, changeIdx uint32) {
	txId := txMsg.TxHash().String()

	_, addrs, _, err := txscript.ExtractPkScriptAddrs(changeOutput.PkScript, w.network)
	if err != nil {
		fmt.Printf("could not extract address script info: %s", err.Error())
		return
	}
	derivationPath := w.getDerivationPathForAddress(addrs[0].String())
	if derivationPath == "" {
		fmt.Printf("no derivation path for address: %v", addrs[0].String())
		return
	}
	value := btcutil.Amount(changeOutput.Value)
	script, err := txscript.ParsePkScript(changeOutput.PkScript)
	if err != nil {
		fmt.Printf("error parsing pkScript of change utxo: %v", err)
	}

	changeUTXO := tx.NewUTXO(txId, changeIdx, value, script.Script(), derivationPath)
	_ = w.addUTXO(*changeUTXO)
}
