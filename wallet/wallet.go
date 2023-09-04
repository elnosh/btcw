package wallet

import (
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/elnosh/btcw/tx"
	"github.com/elnosh/btcw/utils"
	bolt "go.etcd.io/bbolt"
)

type (
	address        = string
	derivationPath = string
	chain          int
)

const (
	externalChain chain = iota
	internalChain
)

type Wallet struct {
	db      *bolt.DB
	client  NodeClient
	network *chaincfg.Params
	logger  *slog.Logger

	utxos   []tx.UTXO
	utxoMtx sync.Mutex

	balance    btcutil.Amount
	balanceMtx sync.Mutex

	lastExternalIdx  uint32
	lastInternalIdx  uint32
	lastScannedBlock int64

	// only for external addresses to track when receiving
	addresses map[address]derivationPath

	locked bool
}

func NewWallet(db *bolt.DB, net *chaincfg.Params) *Wallet {
	logger := slog.Default()
	addresses := make(map[address]derivationPath)
	balance := btcutil.Amount(0)

	return &Wallet{db: db, network: net, logger: logger,
		balance: balance, addresses: addresses}
}

func (w *Wallet) setLastExternalIdx(idx uint32) error {
	err := w.updateLastExternalIdx(idx)
	if err != nil {
		return err
	}
	w.lastExternalIdx = idx
	return nil
}

func (w *Wallet) setLastInternalIdx(idx uint32) error {
	err := w.updateLastInternalIdx(idx)
	if err != nil {
		return err
	}
	w.lastInternalIdx = idx
	return nil
}

func (w *Wallet) setLastScannedBlock(height int64) error {
	err := w.updateLastScannedBlock(height)
	if err != nil {
		return err
	}
	w.lastScannedBlock = height
	return nil
}

func (w *Wallet) setBalance(balance btcutil.Amount) error {
	err := w.updateBalanceDB(balance)
	if err != nil {
		return err
	}
	w.balanceMtx.Lock()
	w.balance = balance
	w.balanceMtx.Unlock()
	return nil
}

func (w *Wallet) addUTXO(utxo tx.UTXO) error {
	err := w.saveUTXO(utxo)
	if err != nil {
		return err
	}
	w.utxoMtx.Lock()
	w.utxos = append(w.utxos, utxo)
	w.utxoMtx.Unlock()
	return nil
}

// add key in db and update addresses map with address of key
func (w *Wallet) addKey(derivationPath string, key *KeyPair) error {
	err := w.saveKeyPair(derivationPath, key)
	if err != nil {
		return err
	}
	w.addresses[key.Address] = derivationPath
	return nil
}

// getDecodedKey retrieves the hashed passphrase from db
// and decodes it.
func (w *Wallet) getDecodedKey() ([]byte, error) {
	encodedHash := w.getEncodedHash()
	if encodedHash == nil {
		return nil, errors.New("encoded hash not found")
	}

	_, key, _, err := utils.DecodeHash(string(encodedHash))
	if err != nil {
		return nil, fmt.Errorf("error decoding key: %v", err)
	}

	return key, nil
}

// getDecryptedAccountKey will take a Chain which can be either external
// or internal and return a decrypted chain key
func (w *Wallet) getDecryptedAccountKey(chain chain) ([]byte, error) {
	var encryptedChainKey []byte

	switch chain {
	case externalChain:
		encryptedChainKey = w.getAcct0External()
	case internalChain:
		encryptedChainKey = w.getAcct0Internal()
	default:
		return nil, errors.New("invalid chain value")
	}

	passKey, err := w.getDecodedKey()
	if err != nil {
		return nil, err
	}

	decryptedChainKey, err := utils.Decrypt(encryptedChainKey, passKey)
	if err != nil {
		return nil, err
	}

	return decryptedChainKey, nil
}

// getPrivateKeyForUTXO returns the private key in WIF that can sign
// or spend that UTXO
func (w *Wallet) getPrivateKeyForUTXO(utxo tx.UTXO) (*btcutil.WIF, error) {
	kp := w.getKeyPair(utxo.DerivationPath)
	if kp == nil {
		return nil, fmt.Errorf("key for UTXO not found")
	}

	passKey, err := w.getDecodedKey()
	if err != nil {
		return nil, err
	}

	wifStr, err := utils.Decrypt(kp.EncryptedPrivateKey, passKey)
	if err != nil {
		return nil, err
	}

	wif, err := btcutil.DecodeWIF(string(wifStr))
	if err != nil {
		return nil, fmt.Errorf("error decoding wif: %s", err.Error())
	}

	return wif, nil
}

func (w *Wallet) lock() {
	w.locked = true
}

func (w *Wallet) unlock(duration time.Duration) {
	w.locked = false

	go func() {
		time.Sleep(duration)
		w.lock()
	}()
}

func (w *Wallet) LogInfo(format string, v ...any) {
	msg := fmt.Sprintf(format, v...)
	w.logger.Info(msg)
}

func (w *Wallet) LogError(format string, v ...any) {
	msg := fmt.Sprintf(format, v...)
	w.logger.Error(msg)
}
