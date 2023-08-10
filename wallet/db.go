package wallet

import (
	"fmt"

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

// init bucket with hashed passphrase
func (w *Wallet) InitAuthBucket(encodedHash string) error {
	return w.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucket([]byte(authBucket))
		if err != nil {
			return err
		}
		return b.Put([]byte(encodedHashKey), []byte(encodedHash))
	})
}

func (w *Wallet) InitUTXOBucket() error {
	return w.db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucket([]byte(utxosBucket))
		if err != nil {
			return err
		}
		return nil
	})
}

func (w *Wallet) InitKeysBucket() error {
	return w.db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucket([]byte(keysBucket))
		if err != nil {
			return err
		}
		return nil
	})
}

func (w *Wallet) InitWalletMetadataBucket(seed []byte, encodedHash string) error {
	master, acct0ext, acct0int, err := DeriveHDKeys(seed, encodedHash)
	if err != nil {
		return err
	}

	_, key, _, err := DecodeKey(encodedHash)
	if err != nil {
		return err
	}

	encryptedMaster, encryptedAcct0ext, encryptedAcct0int, err := EncryptHDKeys(key, master, acct0ext, acct0int)

	return w.db.Update(func(tx *bolt.Tx) error {
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
	})
}

func (w Wallet) GetEncodedHash() []byte {
	var encodedHash []byte
	w.db.View(func(tx *bolt.Tx) error {
		walletMetadata := tx.Bucket([]byte(authBucket))
		encodedHash = walletMetadata.Get([]byte(encodedHashKey))
		return nil
	})
	return encodedHash
}

func (w Wallet) GetAcct0Ext() []byte {
	var encryptedAcct0ext []byte
	w.db.View(func(tx *bolt.Tx) error {
		walletMetadata := tx.Bucket([]byte(walletMetadataBucket))
		encryptedAcct0ext = walletMetadata.Get([]byte(account0ExternelKey))
		return nil
	})
	return encryptedAcct0ext
}

func (w Wallet) increaseLastExternalIdx() error {
	return w.db.Update(func(tx *bolt.Tx) error {
		walletMetadata := tx.Bucket([]byte(walletMetadataBucket))
		err := walletMetadata.Put([]byte(lastExternalIdxKey), utils.Uint32ToBytes(w.lastExternalIdx))
		return err
	})
}

func (w Wallet) saveExternalKeyPair(keypair []byte) error {
	return w.db.Update(func(tx *bolt.Tx) error {
		keysb := tx.Bucket([]byte(keysBucket))
		derivationPath := fmt.Sprintf("m/44'/1'/0'/0/%d", w.lastExternalIdx)
		err := keysb.Put([]byte(derivationPath), keypair)
		return err
	})
}
