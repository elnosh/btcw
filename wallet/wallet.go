package wallet

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/txscript"
	"github.com/elnosh/btcw/tx"
	bolt "go.etcd.io/bbolt"
)

type (
	address        = string
	derivationPath = string
	Chain          int
)

const (
	externalChain Chain = iota
	internalChain
)

type Wallet struct {
	db     *bolt.DB
	client *rpcclient.Client

	utxos            []tx.UTXO
	balance          btcutil.Amount
	lastExternalIdx  uint32
	lastInternalIdx  uint32
	lastScannedBlock int64

	addresses map[address]derivationPath
}

// getDecryptedAccountKey will take a Chain which can be either external
// or internal and return a decrypted chain key
func (w *Wallet) getDecryptedAccountKey(chain Chain) ([]byte, error) {
	var encryptedChainKey []byte

	switch chain {
	case externalChain:
		encryptedChainKey = w.getAcct0External()
	case internalChain:
		encryptedChainKey = w.getAcct0Internal()
	default:
		return nil, errors.New("invalid chain value")
	}

	passKey, err := w.GetDecodedKey()
	if err != nil {
		return nil, err
	}

	decryptedChainKey, err := Decrypt(encryptedChainKey, passKey)
	if err != nil {
		return nil, err
	}

	return decryptedChainKey, nil
}

func (w *Wallet) setLastExternalIdx(idx uint32) error {
	err := w.updateLastExternalIdx(idx)
	if err != nil {
		return err
	}
	w.lastExternalIdx = idx
	return nil
}

func (w *Wallet) setLastInternalIdx(idx uint32) error {
	err := w.updateLastInternalIdx(idx)
	if err != nil {
		return err
	}
	w.lastInternalIdx = idx
	return nil
}

func (w *Wallet) setLastScannedBlock(height int64) error {
	err := w.updateLastScannedBlock(height)
	if err != nil {
		return err
	}
	w.lastScannedBlock = height
	return nil
}

func (w *Wallet) setBalance(balance btcutil.Amount) error {
	err := w.updateBalanceDB(balance)
	if err != nil {
		return err
	}
	w.balance = balance
	return nil
}

func (w *Wallet) addUTXO(utxo *tx.UTXO) error {
	err := w.saveUTXO(utxo)
	if err != nil {
		return err
	}
	w.utxos = append(w.utxos, *utxo)
	return nil
}

func ScanForNewBlocks(ctx context.Context, wallet *Wallet, errChan chan error) {
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
					log.Printf("error getting block count: %s", err.Error())
					errChan <- err
					return
				}
				err = checkBlocks(wallet, height)
				if err != nil {
					fmt.Printf("error checking blocks: %s", err.Error())
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
		// get block info
		block, err := wallet.client.GetBlockVerboseTx(nextBlockHash)
		if err != nil {
			return fmt.Errorf("error getting block: %s", err.Error())
		}

		txsInBlock := block.RawTx
		for _, rawTx := range txsInBlock {
			for _, vout := range rawTx.Vout {
				scriptAsm, err := hex.DecodeString(vout.ScriptPubKey.Hex)
				if err != nil {
					return fmt.Errorf("error decoding hex script: %s", err.Error())
				}

				// this will extract the address from the script
				class, addrs, _, err := txscript.ExtractPkScriptAddrs(scriptAsm, &chaincfg.SimNetParams)
				if err != nil {
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
						utxo := tx.NewUTXO(rawTx.Txid, vout.N, vout.Value, vout.ScriptPubKey.Hex, path)
						if err := wallet.addUTXO(utxo); err != nil {
							return fmt.Errorf("error adding new UTXO: %s", err.Error())
						}

						amt, err := btcutil.NewAmount(vout.Value)
						if err != nil {
							return fmt.Errorf("error getting tx amount: %s", err.Error())
						}

						balance := wallet.balance + amt
						if err := wallet.setBalance(balance); err != nil {
							return fmt.Errorf("error setting wallet balance: %s", err.Error())
						}
					}
				}

			}
		}
		// increase last scanned block
		if err := wallet.setLastScannedBlock(wallet.lastScannedBlock + 1); err != nil {
			return fmt.Errorf("error setting last scanned block: %s", err.Error())
		}
	}
	return nil
}
