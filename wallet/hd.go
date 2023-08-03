package wallet

import (
	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"
)

func DeriveHDKeys(seed []byte, encodedPass string) (master, acct0 *hdkeychain.ExtendedKey, err error) {
	// master node
	// path: m
	master, err = hdkeychain.NewMaster(seed, &chaincfg.SimNetParams)
	if err != nil {
		return nil, nil, err
	}

	// path: m/44'
	bip44, err := master.Derive(hdkeychain.HardenedKeyStart + 44)
	if err != nil {
		return nil, nil, err
	}

	// Bitcoin Testnet - path: m/44'/1'
	ctype, err := bip44.Derive(hdkeychain.HardenedKeyStart + 1)
	if err != nil {
		return nil, nil, err
	}

	// first account - path: m/44'/1'/0'
	acct0, err = ctype.Derive(hdkeychain.HardenedKeyStart + 0)
	if err != nil {
		return nil, nil, err
	}

	return master, acct0, nil
}
