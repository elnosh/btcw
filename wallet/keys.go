package wallet

import (
	"fmt"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"
)

type KeyPair struct {
	PublicKey           []byte `json:"publicKey"`  // serialized
	EncryptedPrivateKey []byte `json:"privateKey"` // WIF
}

func NewKeyPair(extendedKey *hdkeychain.ExtendedKey, key []byte) (*KeyPair, error) {
	// convert extended key to btcec private key
	privateKey, err := extendedKey.ECPrivKey()
	if err != nil {
		return nil, err
	}

	// convert private key -> wif
	wif, err := btcutil.NewWIF(privateKey, &chaincfg.SimNetParams, true)
	if err != nil {
		return nil, err
	}

	// encrypt wif for storage
	encryptedWIF, err := Encrypt([]byte(wif.String()), key)
	if err != nil {
		return nil, fmt.Errorf("error encrypting private key: %v", err)
	}

	// get serialized compressed public key from private key
	serializedPubKey := wif.SerializePubKey()
	keyPair := &KeyPair{PublicKey: serializedPubKey, EncryptedPrivateKey: encryptedWIF}
	return keyPair, nil
}
