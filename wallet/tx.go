package wallet

import (
	"encoding/hex"
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
	txOut, err := tx.CreateTxOut(address, amountToSend)
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
		changeTxOut, err := tx.CreateTxOut(newInternalKey.Address, changeAmount)
		if err != nil {
			return nil, err
		}
		rawTx.AddTxOut(changeTxOut)

		// calculate estimate fee for tx
		size := rawTx.SerializeSize()
		kbSize := int64(size) / 1000
		estimateFee := kbSize * int64(feeRate)

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

		prevScript, err := hex.DecodeString(utxo.ScriptPubKey)
		if err != nil {
			return fmt.Errorf("could not decode scriptPubKey hex: %s", err.Error())
		}

		scriptSig, err := txscript.SignatureScript(tx, i, prevScript,
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
		prevScript, err := hex.DecodeString(utxo.ScriptPubKey)
		if err != nil {
			return fmt.Errorf("could not decode scriptPubKey hex: %s", err.Error())
		}

		vm, err := txscript.NewEngine(prevScript, tx, i, txscript.StandardVerifyFlags, nil, nil, int64(utxo.Value), nil)
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
