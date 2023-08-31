package wallet

import (
	"errors"
	"fmt"

	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/elnosh/btcw/utils"
)

// DeriveHDKeys derives the keys for initial HD wallet setup - BIP-44
// that will be external and internal chain for first account
// external chain path: m/44'/1'/0'/0
// internal chain path: m/44'/1'/0'/1
func DeriveHDKeys(seed []byte, net *chaincfg.Params) (master, acct0ext,
	acct0int *hdkeychain.ExtendedKey, err error) {
	// master node
	// path: m
	master, err = hdkeychain.NewMaster(seed, net)
	if err != nil {
		return nil, nil, nil, err
	}

	if !master.IsPrivate() {
		return nil, nil, nil, errors.New("error deriving keys")
	}

	// path: m/44'
	bip44, err := master.Derive(hdkeychain.HardenedKeyStart + 44)
	if err != nil {
		return nil, nil, nil, err
	}

	// Bitcoin Testnet - path: m/44'/1'
	ctype, err := bip44.Derive(hdkeychain.HardenedKeyStart + 1)
	if err != nil {
		return nil, nil, nil, err
	}

	// first account - path: m/44'/1'/0'
	acct0, err := ctype.Derive(hdkeychain.HardenedKeyStart + 0)
	if err != nil {
		return nil, nil, nil, err
	}

	// external chain of account 0 - path: m/44'/1'/0'/0
	acct0ext, err = acct0.Derive(0)
	if err != nil {
		return nil, nil, nil, err
	}

	// internal chain of account 0 - path: m/44'/1'/0'/1
	acct0int, err = acct0.Derive(1)
	if err != nil {
		return nil, nil, nil, err
	}

	return master, acct0ext, acct0int, nil
}

// EncryptHDKeys encrypts the HD keys derived from DeriveHDKeys
func EncryptHDKeys(key []byte, master, acct0ext, acct0int *hdkeychain.ExtendedKey) (encryptedMaster,
	encryptedAcct0ext, encryptedAcct0int []byte, err error) {
	encryptedMaster, err = utils.Encrypt([]byte(master.String()), key)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error encrypting key: %v", err)
	}

	encryptedAcct0ext, err = utils.Encrypt([]byte(acct0ext.String()), key)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error encrypting key: %v", err)
	}

	encryptedAcct0int, err = utils.Encrypt([]byte(acct0int.String()), key)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error encrypting key: %v", err)
	}

	return encryptedMaster, encryptedAcct0ext, encryptedAcct0int, nil
}

// DeriveNextHDKey will derive the next child key from fromAcctKey
// which could be either for external or internal chain
func DeriveNextHDKey(fromAcctKey []byte, idx uint32) (*hdkeychain.ExtendedKey, error) {
	acctExtendedKey, err := hdkeychain.NewKeyFromString(string(fromAcctKey))
	if err != nil {
		return nil, fmt.Errorf("error deriving new key: %v", err)
	}

	newKey, err := acctExtendedKey.Derive(idx)
	if err != nil {
		return nil, fmt.Errorf("error deriving new key: %v", err)
	}

	return newKey, nil
}
