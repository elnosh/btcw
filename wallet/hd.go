package wallet

import (
	"errors"
	"fmt"

	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/elnosh/btcw/utils"
)

// derive keys for initial HD wallet setup - BIP-44
// that will be external and internal chain for first account
// external chain path: m/44'/1'/0'/0
// internal chain path: m/44'/1'/0'/1
func DeriveHDKeys(seed []byte, encodedPass string) (master, acct0ext,
	acct0int *hdkeychain.ExtendedKey, err error) {
	// master node
	// path: m
	master, err = hdkeychain.NewMaster(seed, &chaincfg.SimNetParams)
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

func EncryptHDKeys(key []byte, master, acct0ext, acct0int *hdkeychain.ExtendedKey) (encryptedMaster,
	encryptedAcct0ext, encryptedAcct0int []byte, err error) {
	encryptedMaster, err = utils.Encrypt([]byte(master.String()), key)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error encrypting key: %s", err.Error())
	}

	encryptedAcct0ext, err = utils.Encrypt([]byte(acct0ext.String()), key)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error encrypting key: %s", err.Error())
	}

	encryptedAcct0int, err = utils.Encrypt([]byte(acct0int.String()), key)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error encrypting key: %s", err.Error())
	}

	return encryptedMaster, encryptedAcct0ext, encryptedAcct0int, nil
}

// fromAcctKey which could be either external or internal, derive next key
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
