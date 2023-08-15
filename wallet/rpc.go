package wallet

import "errors"

func (w *Wallet) GetBalance() int64 {
	return w.balance
}

func (w *Wallet) GetNewAddress() (string, error) {
	// get account_0_external
	encryptedAcct0ext := w.getAcct0Ext()
	if encryptedAcct0ext == nil {
		return "", errors.New("account 0 external not found")
	}

	passKey, err := w.GetDecodedKey()
	if err != nil {
		return "", err
	}

	acct0ext, err := Decrypt(encryptedAcct0ext, passKey)
	if err != nil {
		return "", err
	}

	// derive the next key
	newKey, err := DeriveNextExternalKey(acct0ext, w.lastExternalIdx+1)
	if err != nil {
		return "", err
	}

	// update lastExternalIdx value in db and wallet struct
	newIdx := w.lastExternalIdx + 1
	err = w.setLastExternalIdx(newIdx)
	if err != nil {
		return "", err
	}

	// get new public/private key pair
	// public key is serialized in compressed format
	// private key is encrypted WIF
	keyPair, err := NewKeyPair(newKey, passKey)
	if err != nil {
		return "", err
	}

	// save key pair in db
	err = w.saveExternalKeyPair(keyPair)
	if err != nil {
		return "", err
	}

	return keyPair.Address, nil
}
