package tx

import (
	"errors"
	"fmt"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
)

var (
	ErrInsufficientAmount = errors.New("not enough value in utxos to fulfill amount")
)

// CreateTxIn returns a new wire.TxIn from the utxo referenced
func CreateTxIn(previousUtxo UTXO) (*wire.TxIn, error) {
	prevTxHash, err := chainhash.NewHashFromStr(previousUtxo.TxID)
	if err != nil {
		return nil, fmt.Errorf("error getting previous tx hash: %s", err.Error())
	}
	prevOutPoint := wire.NewOutPoint(prevTxHash, previousUtxo.VoutIdx)
	txIn := wire.NewTxIn(prevOutPoint, nil, nil)
	return txIn, nil
}

// CreateTxOut returns a wire.TxOut with the script to pay the amount
// to the address passed
func CreateTxOut(address string, amount btcutil.Amount) (*wire.TxOut, error) {
	addr, err := btcutil.DecodeAddress(address, &chaincfg.SimNetParams)
	if err != nil {
		return nil, fmt.Errorf("error decoding address for tx out: %s", err.Error())
	}

	script, err := txscript.PayToAddrScript(addr)
	if err != nil {
		return nil, fmt.Errorf("error creating tx out script: %s", err.Error())
	}

	txOut := wire.NewTxOut(int64(amount), script)
	return txOut, nil
}
