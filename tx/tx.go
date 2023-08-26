package tx

import (
	"fmt"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
)

// CreateTxIn returns a new wire.TxIn from the utxo referenced
func CreateTxIn(previousUtxo UTXO) (*wire.TxIn, error) {
	prevTxHash, err := chainhash.NewHashFromStr(previousUtxo.TxID)
	if err != nil {
		return nil, fmt.Errorf("chainhash.NewHashFromStr: %v", err)
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
		return nil, fmt.Errorf("btcutil.DecodeAddress: %v", err)
	}

	script, err := txscript.PayToAddrScript(addr)
	if err != nil {
		return nil, fmt.Errorf("txscript.PayToAddrScript: %v", err)
	}

	txOut := wire.NewTxOut(int64(amount), script)
	return txOut, nil
}
