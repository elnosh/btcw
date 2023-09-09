package wallet

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/elnosh/btcw/tx"
)

// scanMissingBlocks will look at the last scanned block and the current
// height of the blockchain and scan any missing blocks if needed
func (w *Wallet) scanMissingBlocks() {
	height, err := w.client.GetBlockCount()
	if err != nil {
		w.LogError("error scanning blockchain - could not get block count: %v", err)
		return
	}

	for w.lastScannedBlock < height {
		nextBlockHash, err := w.client.GetBlockHash(w.lastScannedBlock + 1)
		if err != nil {
			w.LogError("error scanning blockchain - could not get block hash: %v", err)
		}

		w.scanBlock(nextBlockHash)
	}
	w.LogInfo("Finished scanning. Synced with blockchain at height: %v", w.lastScannedBlock)
}

// scanBlockTxs scans the new block received for addresses owned by
// wallet and adds UTXOs to wallet and updates balance.
// This is called by btcd notification handler setup for when
// new blocks are added to the blockchain
func (w *Wallet) scanBlockTxs(blockHash string, txsInBlock []*btcutil.Tx) {
	for _, txb := range txsInBlock {
		for voutIdx, txOut := range txb.MsgTx().TxOut {
			script, err := txscript.ParsePkScript(txOut.PkScript)
			if err != nil {
				w.LogError("error scanning block - could not parse tx pkScript: %v", err)
			}

			addr, err := script.Address(w.network)
			if err != nil {
				w.LogError("error scanning block - could not get address: %v", err)
			}

			addrStr := addr.String()
			path, ok := w.addresses[addrStr]
			// if ok, output found that sends to address owned by wallet
			if ok {
				w.LogInfo("found new receiving transaction in block %s", blockHash)
				value := btcutil.Amount(txOut.Value)
				txid := txb.Hash().String()

				utxo := tx.NewUTXO(txid, uint32(voutIdx), value, script.Script(), path)
				if err := w.addUTXO(*utxo); err != nil {
					w.LogError("error adding receiving UTXO to wallet: %v", err)
				}

				balance := w.balance + utxo.Value
				if err := w.setBalance(balance); err != nil {
					w.LogError("error updating balance for new receiving UTXO: %v", err)
				}

				w.LogInfo("added new transaction %s to wallet", txid)
			}
		}
	}
	w.setLastScannedBlock(w.lastScannedBlock + 1)
}

// scanForNewBlocks used when node is bitcoin core
func scanForNewBlocks(ctx context.Context, wallet *Wallet, errChan chan error) {
	wallet.LogInfo("Scanning for new blocks")
	go func(ctx context.Context) {
		ticker := time.NewTicker(time.Second * 30)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				errChan <- nil
				return
			case <-ticker.C:
				height, err := wallet.client.GetBlockCount()
				if err != nil {
					errChan <- err
					return
				}
				// if there are any new blocks, scan them
				for wallet.lastScannedBlock < height {
					nextBlockHash, err := wallet.client.GetBlockHash(wallet.lastScannedBlock + 1)
					if err != nil {
						errChan <- fmt.Errorf("error getting block hash: %v", err)
					}

					wallet.scanBlock(nextBlockHash)
				}
			}
		}
	}(ctx)
}

func (w *Wallet) scanBlock(blockHash *chainhash.Hash) {
	// get block info
	block, err := w.client.GetBlockVerboseTx(blockHash)
	if err != nil {
		w.LogError("error getting block: %v", err)
		return
	}

	// there is a difference between btcd and bitcoin core
	// in the []TxRawResult returned from GetBlockVerboseTx call.
	// if node is btcd use RawTx field, if core then use Tx field
	var txsInBlock []btcjson.TxRawResult
	switch v := w.client.(type) {
	case *BtcdClient:
		txsInBlock = block.RawTx
	case *BitcoinCoreClient:
		txsInBlock = block.Tx
	default:
		w.LogError("invalid client type: %T", v)
	}

	for _, rawTx := range txsInBlock {
		for _, vout := range rawTx.Vout {
			script, err := hex.DecodeString(vout.ScriptPubKey.Hex)
			if err != nil {
				w.LogError("error decoding hex script: %v", err)
				return
			}

			// this will extract the address from the script
			class, addrs, _, err := txscript.ExtractPkScriptAddrs(script, w.network)
			if err != nil {
				w.LogError("error extractring address script info: %v", err)
				return
			}

			// only handling pub key hash for now
			if class == txscript.PubKeyHashTy {
				// check if address extracted from script is in wallet
				addr := addrs[0].String()
				path, ok := w.addresses[addr]
				// if match is found
				// add UTXO and update wallet balance
				if ok {
					w.LogInfo("found new receiving transaction in block %s", block.Hash)
					utxoAmount, err := btcutil.NewAmount(vout.Value)
					if err != nil {
						w.LogError("error getting tx amount: %v", err)
						return
					}

					utxo := tx.NewUTXO(rawTx.Txid, vout.N, utxoAmount, script, path)
					if err := w.addUTXO(*utxo); err != nil {
						w.LogError("error adding new UTXO: %v", err)
						return
					}

					balance := w.balance + utxoAmount
					if err := w.setBalance(balance); err != nil {
						w.LogError("error setting wallet balance: %v", err)
						return
					}
					w.LogInfo("added new transaction %s to wallet", rawTx.Txid)
				}
			}

		}
	}
	// increase last scanned block
	w.setLastScannedBlock(w.lastScannedBlock + 1)
}
