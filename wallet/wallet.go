package wallet

import (
	"encoding/json"
	"errors"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/elnosh/btcw/tx"
	bolt "go.etcd.io/bbolt"
)

type Wallet struct {
	db     *bolt.DB
	client *rpcclient.Client

	utxos            []tx.UTXO
	balance          int64
	lastExternalIdx  uint32
	lastInternalIdx  uint32
	lastScannedBlock int64
}

func NewWallet(db *bolt.DB) *Wallet {
	return &Wallet{db: db}
}

func (w *Wallet) Setdb(db *bolt.DB) {
	w.db = db
}

func (w *Wallet) GetBalance() int64 {
	return w.balance
}

func (w *Wallet) GetNewAddress() (string, error) {
	// get account_0_external
	encryptedAcct0ext := w.GetAcct0Ext()
	if encryptedAcct0ext == nil {
		return "", errors.New("account 0 external not found")
	}

	key, err := w.GetDecodedKey()
	if err != nil {
		return "", err
	}

	acct0ext, err := Decrypt(encryptedAcct0ext, key)
	if err != nil {
		return "", err
	}

	// derive the next key
	newKey, err := DeriveNextExternalKey(acct0ext, w.lastExternalIdx+1)
	if err != nil {
		return "", err
	}

	// update lastExternalIdx value in db and wallet struct
	w.lastExternalIdx += 1
	err = w.increaseLastExternalIdx()
	if err != nil {
		return "", err
	}

	// get new public/private key pair
	// public key is serialized in compressed format
	// private key is encrypted WIF
	keyPair, err := NewKeyPair(newKey, key)
	if err != nil {
		return "", err
	}

	jsonKeyPair, err := json.Marshal(keyPair)
	if err != nil {
		return "", err
	}

	// save json encoding of key pair in db
	err = w.saveExternalKeyPair(jsonKeyPair)
	if err != nil {
		return "", err
	}

	// convert pub key -> addressPubKeyHash to return address as string
	addrPubKey, err := btcutil.NewAddressPubKey(keyPair.PublicKey, &chaincfg.SimNetParams)
	if err != nil {
		return "", err
	}

	addrPubKeyHash := addrPubKey.AddressPubKeyHash().String()
	return addrPubKeyHash, nil
}
