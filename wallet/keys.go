package wallet

import (
	"fmt"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/elnosh/btcw/utils"
)

type KeyPair struct {
	PublicKey           []byte `json:"publicKey"`  // serialized
	EncryptedPrivateKey []byte `json:"privateKey"` // WIF
	PublicKeyHash       []byte `json:"publicKeyHash"`
	Address             string `json:"address"`
}

// generate a new key pair from the extended key
// public key is serialized in compressed format
// private key is encrypted in WIF format
func (w *Wallet) newKeyPair(extendedKey *hdkeychain.ExtendedKey) (*KeyPair, error) {
	// convert extended key to btcec private key
	privateKey, err := extendedKey.ECPrivKey()
	if err != nil {
		return nil, err
	}

	// convert private key -> wif
	wif, err := btcutil.NewWIF(privateKey, w.network, true)
	if err != nil {
		return nil, err
	}

	passKey, err := w.getDecodedKey()
	if err != nil {
		return nil, err
	}

	// encrypt wif for storage
	encryptedWIF, err := utils.Encrypt([]byte(wif.String()), passKey)
	if err != nil {
		return nil, fmt.Errorf("error encrypting private key: %v", err.Error())
	}

	// get serialized compressed public key from private key
	serializedPubKey := wif.SerializePubKey()

	// derive public key hash and address string from the
	// serialized public key
	addrPubKey, err := btcutil.NewAddressPubKey(serializedPubKey, w.network)
	if err != nil {
		return nil, fmt.Errorf("error deriving address pub key: %v", err.Error())
	}
	addrPubKeyHash := addrPubKey.AddressPubKeyHash()
	pubkeyHash := addrPubKeyHash.ScriptAddress()
	addr := addrPubKeyHash.String()

	keyPair := &KeyPair{PublicKey: serializedPubKey, EncryptedPrivateKey: encryptedWIF,
		PublicKeyHash: pubkeyHash, Address: addr}
	return keyPair, nil
}

func (w *Wallet) generateNewExternalKeyPair() (*KeyPair, error) {
	acct0external, err := w.getDecryptedAccountKey(externalChain)
	if err != nil {
		return nil, err
	}

	// derive the next external key
	childKey, err := DeriveNextHDKey(acct0external, w.lastExternalIdx)
	if err != nil {
		return nil, err
	}

	keyPair, err := w.newKeyPair(childKey)
	if err != nil {
		return nil, err
	}

	// save newly generate key pair with derivation path as key
	derivationPath := fmt.Sprintf("m/44'/1'/0'/0/%d", w.lastExternalIdx)
	err = w.addKey(derivationPath, keyPair)
	if err != nil {
		return nil, err
	}

	// update lastExternalIdx value in db and wallet struct
	newIdx := w.lastExternalIdx + 1
	err = w.setLastExternalIdx(newIdx)
	if err != nil {
		return nil, err
	}

	return keyPair, nil
}

// generateNewInternalKeyPair generates key in internal chain
// for change outputs.
func (w *Wallet) generateNewInternalKeyPair() (*KeyPair, error) {
	acct0internal, err := w.getDecryptedAccountKey(internalChain)
	if err != nil {
		return nil, err
	}

	// derive the next external key
	childKey, err := DeriveNextHDKey(acct0internal, w.lastInternalIdx)
	if err != nil {
		return nil, err
	}

	keyPair, err := w.newKeyPair(childKey)
	if err != nil {
		return nil, err
	}

	derivationPath := fmt.Sprintf("m/44'/1'/0'/1/%d", w.lastInternalIdx)
	err = w.saveKeyPair(derivationPath, keyPair)
	if err != nil {
		return nil, err
	}

	// update lastExternalIdx value in db and wallet struct
	newIdx := w.lastInternalIdx + 1
	err = w.setLastInternalIdx(newIdx)
	if err != nil {
		return nil, err
	}

	return keyPair, nil
}
