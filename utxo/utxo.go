package utxo

import "github.com/btcsuite/btcd/btcjson"

type UTXO struct {
	TxOut          *btcjson.GetTxOutResult // response from gettxout call
	Spent          bool
	DerivationPath string // path of the key associated with this utxo
}
