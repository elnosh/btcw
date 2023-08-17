package wallet

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/elnosh/btcw/tx"
	"github.com/elnosh/btcw/utils"
	bolt "go.etcd.io/bbolt"
)

const (
	// bucket names
	authBucket           = "auth"
	utxosBucket          = "utxos"
	keysBucket           = "keys"
	walletMetadataBucket = "wallet_metadata"

	// constant keys
	encodedHashKey      = "encoded_hash"
	balanceKey          = "balance"
	masterSeedKey       = "master_seed"
	account0ExternelKey = "account_0_external"
	account0InternalKey = "account_0_internal"
	lastScannedBlockKey = "last_scanned_block"
	lastExternalIdxKey  = "last_external_idx"
	lastInternalIdxKey  = "last_internal_idx"
)

// create auth, utxos, keys and wallet metadata buckets
func (w *Wallet) InitWalletBuckets(seed []byte, encodedHash string) error {
	return w.db.Update(func(tx *bolt.Tx) error {
		if err := createAuthBucket(tx, encodedHash); err != nil {
			return err
		}
		if err := createUTXOBucket(tx); err != nil {
			return err
		}
		if err := createKeysBucket(tx); err != nil {
			return err
		}

		// derive keys to be stored
		master, acct0ext, acct0int, err := DeriveHDKeys(seed, encodedHash)
		if err != nil {
			return err
		}

		// decoded hash to get passkey
		_, key, _, err := DecodeHash(encodedHash)
		if err != nil {
			return err
		}

		// encrypt derived keys
		encryptedMaster, encryptedAcct0ext, encryptedAcct0int, err := EncryptHDKeys(key, master, acct0ext, acct0int)
		if err != nil {
			return err
		}

		if err := createWalletMetadataBucket(tx, encryptedMaster, encryptedAcct0ext, encryptedAcct0int); err != nil {
			return err
		}
		return nil
	})
}

// create bucket with hashed passphrase
func createAuthBucket(tx *bolt.Tx, encodedHash string) error {
	b, err := tx.CreateBucket([]byte(authBucket))
	if err != nil {
		return err
	}
	return b.Put([]byte(encodedHashKey), []byte(encodedHash))
}

func createUTXOBucket(tx *bolt.Tx) error {
	_, err := tx.CreateBucket([]byte(utxosBucket))
	return err
}

func createKeysBucket(tx *bolt.Tx) error {
	_, err := tx.CreateBucket([]byte(keysBucket))
	return err
}

func createWalletMetadataBucket(tx *bolt.Tx, encryptedMaster, encryptedAcct0ext, encryptedAcct0int []byte) error {
	wallet, err := tx.CreateBucket([]byte(walletMetadataBucket))
	if err != nil {
		return err
	}

	if err = wallet.Put([]byte(balanceKey), utils.Int64ToBytes(0)); err != nil {
		return err
	}

	if err = wallet.Put([]byte(lastScannedBlockKey), utils.Int64ToBytes(0)); err != nil {
		return err
	}

	// set derivation paths needed
	if err = wallet.Put([]byte(masterSeedKey), encryptedMaster); err != nil {
		return err
	}
	if err = wallet.Put([]byte(account0ExternelKey), encryptedAcct0ext); err != nil {
		return err
	}
	if err = wallet.Put([]byte(account0InternalKey), encryptedAcct0int); err != nil {
		return err
	}
	if err = wallet.Put([]byte(lastExternalIdxKey), utils.Uint32ToBytes(0)); err != nil {
		return err
	}
	if err = wallet.Put([]byte(lastInternalIdxKey), utils.Uint32ToBytes(0)); err != nil {
		return err
	}

	return nil
}

func (w Wallet) getEncodedHash() []byte {
	var encodedHash []byte
	w.db.View(func(tx *bolt.Tx) error {
		walletMetadata := tx.Bucket([]byte(authBucket))
		encodedHash = walletMetadata.Get([]byte(encodedHashKey))
		return nil
	})
	return encodedHash
}

func (w Wallet) getBalance() btcutil.Amount {
	var bytes []byte
	w.db.View(func(tx *bolt.Tx) error {
		walletMetadata := tx.Bucket([]byte(walletMetadataBucket))
		bytes = walletMetadata.Get([]byte(balanceKey))
		return nil
	})
	balance := btcutil.Amount(utils.BytesToInt64(bytes))
	return balance
}

func (w Wallet) getLastExternalIdx() uint32 {
	var bytes []byte
	w.db.View(func(tx *bolt.Tx) error {
		walletMetadata := tx.Bucket([]byte(walletMetadataBucket))
		bytes = walletMetadata.Get([]byte(lastExternalIdxKey))
		return nil
	})
	lastExternalIdx := utils.BytesToUint32(bytes)
	return lastExternalIdx
}

func (w Wallet) getLastInternalIdx() uint32 {
	var bytes []byte
	w.db.View(func(tx *bolt.Tx) error {
		walletMetadata := tx.Bucket([]byte(walletMetadataBucket))
		bytes = walletMetadata.Get([]byte(lastInternalIdxKey))
		return nil
	})
	lastInternalIdx := utils.BytesToUint32(bytes)
	return lastInternalIdx
}

func (w Wallet) getLastScannedBlock() int64 {
	var bytes []byte
	w.db.View(func(tx *bolt.Tx) error {
		walletMetadata := tx.Bucket([]byte(walletMetadataBucket))
		bytes = walletMetadata.Get([]byte(lastScannedBlockKey))
		return nil
	})
	lastScannedBlock := utils.BytesToInt64(bytes)
	return lastScannedBlock
}

func (w Wallet) getAcct0Ext() []byte {
	var encryptedAcct0ext []byte
	w.db.View(func(tx *bolt.Tx) error {
		walletMetadata := tx.Bucket([]byte(walletMetadataBucket))
		encryptedAcct0ext = walletMetadata.Get([]byte(account0ExternelKey))
		return nil
	})
	return encryptedAcct0ext
}

func (w *Wallet) updateLastExternalIdx(idx uint32) error {
	if err := w.db.Update(func(tx *bolt.Tx) error {
		walletMetadata := tx.Bucket([]byte(walletMetadataBucket))
		newIdx := utils.Uint32ToBytes(idx)
		err := walletMetadata.Put([]byte(lastExternalIdxKey), newIdx)
		return err
	}); err != nil {
		return fmt.Errorf("error updating last external idx: %s", err.Error())
	}
	return nil
}

func (w *Wallet) updateLastScannedBlock(height int64) error {
	if err := w.db.Update(func(tx *bolt.Tx) error {
		walletMetadata := tx.Bucket([]byte(walletMetadataBucket))
		v := utils.Int64ToBytes(height)
		err := walletMetadata.Put([]byte(lastScannedBlockKey), v)
		return err
	}); err != nil {
		return fmt.Errorf("error updating last scanned block: %s", err.Error())
	}
	return nil
}

func (w *Wallet) saveExternalKeyPair(keypair *KeyPair) error {
	jsonbytes, err := json.Marshal(keypair)
	if err != nil {
		return fmt.Errorf("error marshalling key pair: %s", err.Error())
	}

	if err := w.db.Update(func(tx *bolt.Tx) error {
		keysb := tx.Bucket([]byte(keysBucket))
		derivationPath := fmt.Sprintf("m/44'/1'/0'/0/%d", w.lastExternalIdx)
		err := keysb.Put([]byte(derivationPath), jsonbytes)
		return err
	}); err != nil {
		return fmt.Errorf("error saving external key pair: %s", err.Error())
	}
	return nil
}

func (w *Wallet) saveUTXO(utxo *tx.UTXO) error {
	jsonbytes, err := json.Marshal(utxo)
	if err != nil {
		return fmt.Errorf("error marshalling utxo: %s", err.Error())
	}

	if err := w.db.Update(func(tx *bolt.Tx) error {
		utxosb := tx.Bucket([]byte(utxosBucket))
		idx := strconv.FormatUint(uint64(utxo.VoutIdx), 10)
		key := []byte(utxo.TxID + idx)
		err := utxosb.Put(key, jsonbytes)
		return err
	}); err != nil {
		return fmt.Errorf("error saving utxo: %s", err.Error())
	}
	return nil
}

func (w *Wallet) updateBalanceDB(balance btcutil.Amount) error {
	balancebytes := utils.Int64ToBytes(int64(balance))

	if err := w.db.Update(func(tx *bolt.Tx) error {
		walletMetadata := tx.Bucket([]byte(walletMetadataBucket))
		err := walletMetadata.Put([]byte(balanceKey), balancebytes)
		return err
	}); err != nil {
		return fmt.Errorf("error updating balance: %s", err.Error())
	}
	return nil
}

func (w *Wallet) loadUTXOs() error {
	utxos := make([]tx.UTXO, 0, 100)

	if err := w.db.View(func(dbtx *bolt.Tx) error {
		utxosb := dbtx.Bucket([]byte(utxosBucket))

		c := utxosb.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var utxo tx.UTXO
			if err := json.Unmarshal(v, &utxo); err != nil {
				return fmt.Errorf("error loading UTXOs: %s", err.Error())
			}
			utxos = append(utxos, utxo)
		}
		return nil
	}); err != nil {
		return err
	}
	w.utxos = utxos
	return nil
}

func (w *Wallet) loadAddresses() error {
	if err := w.db.View(func(tx *bolt.Tx) error {
		keysb := tx.Bucket([]byte(keysBucket))

		c := keysb.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var kp KeyPair
			if err := json.Unmarshal(v, &kp); err != nil {
				return fmt.Errorf("error loading addresses: %s", err.Error())
			}
			w.addresses[kp.Address] = string(k)
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}
