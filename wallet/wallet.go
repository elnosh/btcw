package wallet

import (
	"time"

	"github.com/btcsuite/btcd/rpcclient"
	"github.com/elnosh/btcw/tx"
	bolt "go.etcd.io/bbolt"
)

type Wallet struct {
	db     *bolt.DB
	client *rpcclient.Client

	utxos            []tx.UTXO
	balance          int64
	lastExternalIdx  int
	lastInternalIdx  int
	lastScannedBlock int
}

// GetBalance method to be exposed by server
func (w *Wallet) GetBalance() int64 {
	return w.balance
}

// check for new blocks in the background, update UTXOs, balance, last fields
func scanForNewBlocks(wallet *Wallet) {
	ticker := time.NewTicker(time.Second * 15)
	quit := make(chan bool)

	go func() {
		for {
			select {
			case <-ticker.C:
				checkBlocks()
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()

	quit <- true
}

func checkBlocks() {

}
