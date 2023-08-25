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
	"github.com/elnosh/btcw/utils"
	bolt "go.etcd.io/bbolt"
)

type (
	address        = string
	derivationPath = string
	chain          int
)

const (
	externalChain chain = iota
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

	addresses map[address]derivationPath // only for external addresses to track when receiving
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

func (w *Wallet) addUTXO(utxo tx.UTXO) error {
	err := w.saveUTXO(utxo)
	if err != nil {
		return err
	}
	w.utxos = append(w.utxos, utxo)
	return nil
}

// add key in db and update addresses map with address of key
func (w *Wallet) addKey(derivationPath string, key *KeyPair) error {
	err := w.saveKeyPair(derivationPath, key)
	if err != nil {
		return err
	}
	w.addresses[key.Address] = derivationPath
	return nil
}

func (w Wallet) getDecodedKey() ([]byte, error) {
	encodedHash := w.getEncodedHash()
	if encodedHash == nil {
		return nil, errors.New("encoded hash not found")
	}

	// decode hash to get key
	_, key, _, err := utils.DecodeHash(string(encodedHash))
	if err != nil {
		return nil, fmt.Errorf("error decoding key: %v", err)
	}

	return key, nil
}

// getDecryptedAccountKey will take a Chain which can be either external
// or internal and return a decrypted chain key
func (w *Wallet) getDecryptedAccountKey(chain chain) ([]byte, error) {
	var encryptedChainKey []byte

	switch chain {
	case externalChain:
		encryptedChainKey = w.getAcct0External()
	case internalChain:
		encryptedChainKey = w.getAcct0Internal()
	default:
		return nil, errors.New("invalid chain value")
	}

	passKey, err := w.getDecodedKey()
	if err != nil {
		return nil, err
	}

	decryptedChainKey, err := utils.Decrypt(encryptedChainKey, passKey)
	if err != nil {
		return nil, err
	}

	return decryptedChainKey, nil
}

// getPrivateKeyForUTXO returns the private key in WIF that can sign
// or spend that UTXO
func (w *Wallet) getPrivateKeyForUTXO(utxo tx.UTXO) (*btcutil.WIF, error) {
	kp := w.getKeyPair(utxo.DerivationPath)
	if kp == nil {
		return nil, fmt.Errorf("key for UTXO not found")
	}

	passKey, err := w.getDecodedKey()
	if err != nil {
		return nil, err
	}

	wifStr, err := utils.Decrypt(kp.EncryptedPrivateKey, passKey)
	if err != nil {
		return nil, err
	}

	wif, err := btcutil.DecodeWIF(string(wifStr))
	if err != nil {
		return nil, fmt.Errorf("error decoding wif: %s", err.Error())
	}

	return wif, nil
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
						utxoAmount, err := btcutil.NewAmount(vout.Value)
						if err != nil {
							return fmt.Errorf("error getting tx amount: %s", err.Error())
						}

						utxo := tx.NewUTXO(rawTx.Txid, vout.N, utxoAmount, vout.ScriptPubKey.Hex, path)
						if err := wallet.addUTXO(*utxo); err != nil {
							return fmt.Errorf("error adding new UTXO: %s", err.Error())
						}

						balance := wallet.balance + utxoAmount
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
