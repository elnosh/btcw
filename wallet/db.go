package wallet

import (
	"github.com/elnosh/btcw/utils"
	bolt "go.etcd.io/bbolt"
)

// func InitWalletDB(db *bolt.DB, encodedHash string, ) {

// }

// init bucket with hashed passphrase
func InitAuthBucket(db *bolt.DB, encodedHash string) error {
	return db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucket([]byte("auth"))
		if err != nil {
			return err
		}
		return b.Put([]byte("encodedhash"), []byte(encodedHash))
	})
}

func InitUTXOBucket(db *bolt.DB) error {
	return db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucket([]byte("utxos"))
		if err != nil {
			return err
		}
		return nil
	})
}

func InitWalletMetadataBucket(db *bolt.DB, seed []byte, encodedHash string) error {
	master, acct0ext, acct0int, err := DeriveHDKeys(seed, encodedHash)
	if err != nil {
		return err
	}

	_, key, _, err := DecodeKey(encodedHash)
	if err != nil {
		return err
	}

	encryptedMaster, encryptedAcct0ext, encryptedAcct0int, err := EncryptHDKeys(key, master, acct0ext, acct0int)

	return db.Update(func(tx *bolt.Tx) error {
		wallet, err := tx.CreateBucket([]byte("wallet_metadata"))
		if err != nil {
			return err
		}

		// set balance
		if err = wallet.Put([]byte("balance"), utils.Int64ToBytes(0)); err != nil {
			return err
		}

		// set derivation paths needed
		if err = wallet.Put([]byte("master_seed"), encryptedMaster); err != nil {
			return err
		}
		if err = wallet.Put([]byte("account_0_external"), encryptedAcct0ext); err != nil {
			return err
		}
		if err = wallet.Put([]byte("account_0_internal"), encryptedAcct0int); err != nil {
			return err
		}
		if err = wallet.Put([]byte("last_external_idx"), utils.Int64ToBytes(0)); err != nil {
			return err
		}
		if err = wallet.Put([]byte("last_internal_idx"), utils.Int64ToBytes(0)); err != nil {
			return err
		}

		return nil
	})
}
