package tx

type UTXO struct {
	TxID           string
	VoutIdx        uint32
	Value          float64
	Spent          bool
	DerivationPath string // path of the key associated with this utxo
}

func NewUTXO(txid string, voutIdx uint32, value float64, path string) *UTXO {
	return &UTXO{TxID: txid, VoutIdx: voutIdx, Value: value, Spent: false, DerivationPath: path}
}
