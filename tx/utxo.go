package tx

type UTXO struct {
	TxID           string
	VoutIdx        uint32
	Value          float64
	ScriptPubKey   string
	Spent          bool
	DerivationPath string // path of the key associated with this utxo
}

func NewUTXO(txid string, voutIdx uint32, value float64, script, path string) *UTXO {
	return &UTXO{TxID: txid, VoutIdx: voutIdx, Value: value, ScriptPubKey: script,
		Spent: false, DerivationPath: path}
}
