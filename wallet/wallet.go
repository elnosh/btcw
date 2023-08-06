package wallet

import (
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/elnosh/btcw/utxo"
	bolt "go.etcd.io/bbolt"
)

type Wallet struct {
	db     *bolt.DB
	client *rpcclient.Client

	UTXOs            []utxo.UTXO
	Balance          int64
	LastExternalIdx  int
	LastInternalIdx  int
	LastScannedBlock int
}
