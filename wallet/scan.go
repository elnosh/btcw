package wallet

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/elnosh/btcw/tx"
)

// scanBlockTxs scans the new block received for addresses owned by
// wallet and adds UTXOs to wallet and updates balance.
// This is called by btcd notification handler setup for when
// new blocks are added to the blockchain
func (w *Wallet) scanBlockTxs(blockHash string, txsInBlock []*btcutil.Tx) {
	w.LogInfo("scanning block received")
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

			path, ok := w.addresses[addr.String()]
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
	w.LogInfo("finished scanning block")
}

// ScanForNewBlocks used when node is bitcoin core
func ScanForNewBlocks(ctx context.Context, wallet *Wallet, errChan chan error) {
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
				err = checkBlocks(wallet, height)
				if err != nil {
					errChan <- err
					return
				}
			}
		}
	}(ctx)
}

// check for new blocks
// it will look for UTXOs for addresses owned by wallet
// if finds any, it will update wallet UTXOs, balance, last fields
func checkBlocks(wallet *Wallet, height int64) error {
	for wallet.lastScannedBlock < height {
		// get hash of next block to scan
		nextBlockHash, err := wallet.client.GetBlockHash(wallet.lastScannedBlock + 1)
		if err != nil {
			return fmt.Errorf("error getting block hash: %s", err.Error())
		}

		err = scanBlock(wallet, nextBlockHash)
		if err != nil {

		}
	}
	return nil
}

func scanBlock(wallet *Wallet, blockHash *chainhash.Hash) error {
	// get block info
	block, err := wallet.client.GetBlockVerboseTx(blockHash)
	if err != nil {
		wallet.LogError("error getting block: %v", err)
		return fmt.Errorf("error getting block: %s", err.Error())
	}

	txsInBlock := block.Tx
	for _, rawTx := range txsInBlock {
		for _, vout := range rawTx.Vout {
			script, err := hex.DecodeString(vout.ScriptPubKey.Hex)
			if err != nil {
				wallet.LogError("error decoding hex script: %s", err.Error())
				return fmt.Errorf("error decoding hex script: %s", err.Error())
			}

			// this will extract the address from the script
			class, addrs, _, err := txscript.ExtractPkScriptAddrs(script, wallet.network)
			if err != nil {
				wallet.LogError("error extractring address script info: %s", err.Error())
				return fmt.Errorf("error extractring address script info: %s", err.Error())
			}

			// only handling pub key hash for now
			if class == txscript.PubKeyHashTy {
				// check if address extracted from script is in wallet
				addr := addrs[0].String()
				path, ok := wallet.addresses[addr]
				// if match is found
				// add UTXO and update wallet balance
				if ok {
					wallet.LogInfo("found new receiving transaction in block %s", block.Hash)
					utxoAmount, err := btcutil.NewAmount(vout.Value)
					if err != nil {
						wallet.LogError("error getting tx amount: %s", err.Error())
						return fmt.Errorf("error getting tx amount: %s", err.Error())
					}

					utxo := tx.NewUTXO(rawTx.Txid, vout.N, utxoAmount, script, path)
					if err := wallet.addUTXO(*utxo); err != nil {
						wallet.LogError("error adding new UTXO: %s", err.Error())
						return fmt.Errorf("error adding new UTXO: %s", err.Error())
					}

					balance := wallet.balance + utxoAmount
					if err := wallet.setBalance(balance); err != nil {
						wallet.LogError("error setting wallet balance: %s", err.Error())
						return fmt.Errorf("error setting wallet balance: %s", err.Error())
					}
					wallet.LogInfo("added new transaction %s to wallet", rawTx.Txid)
				}
			}

		}
	}
	// increase last scanned block
	if err := wallet.setLastScannedBlock(wallet.lastScannedBlock + 1); err != nil {
		return fmt.Errorf("error setting last scanned block: %s", err.Error())
	}
	return nil
}
