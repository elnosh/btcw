package wallet

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/rpcclient"
	bolt "go.etcd.io/bbolt"
	"golang.org/x/term"
)

var (
	ErrPass            = errors.New("error reading passphrase, please try again")
	ErrWalletNotExists = errors.New("wallet does not exist")
)

func NewWallet(db *bolt.DB) *Wallet {
	return &Wallet{db: db}
}

func CreateWallet() error {
	path := setupWalletDir()
	db, err := bolt.Open(filepath.Join(path, "wallet.db"), 0600, nil)
	if err != nil {
		return errors.New("error setting wallet")
	}

	if walletExists(db) {
		return errors.New("wallet already exists")
	}

	wallet := &Wallet{db: db}
	defer wallet.db.Close()

	// create wallet prompt
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("do you want to create a new wallet? (y/n)")
	input, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal("error reading input, please try again")
	}

	input = strings.ToLower(strings.TrimSpace(input))
	var encodedHash string
	if input == "y" || input == "yes" {
		encodedHash, err = promptPassphrase()
		if err != nil {
			return err
		}
	} else {
		os.Exit(0)
	}

	if err = wallet.InitAuthBucket(encodedHash); err != nil {
		return err
	}

	seed, err := hdkeychain.GenerateSeed(hdkeychain.RecommendedSeedLen)
	if err != nil {
		return err
	}

	fmt.Println("Next will be the master seed. Write it down and store securely. Anyone with access to the seed has access to the funds.")
	fmt.Printf("seed: %x\n", seed)

	if err = wallet.InitWalletMetadataBucket(seed, encodedHash); err != nil {
		return err
	}

	if err = wallet.InitUTXOBucket(); err != nil {
		return err
	}

	if err = wallet.InitKeysBucket(); err != nil {
		return err
	}

	return nil
}

func promptPassphrase() (string, error) {
	fmt.Print("enter passphrase for wallet: \n")
	passphrase, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return "", ErrPass
	}
	fmt.Print("confirm passphrase: \n")
	confirmPassphrase, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return "", ErrPass
	}
	if !bytes.Equal(passphrase, confirmPassphrase) {
		return "", errors.New("passphrases do not match, please try again")
	}

	encodedHash, err := hashPassphrase(confirmPassphrase)
	if err != nil {
		return "", err
	}

	return encodedHash, nil
}

func setupWalletDir() string {
	homedir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	path := filepath.Join(homedir, ".btcw", "wallet")
	err = os.MkdirAll(path, 0700)
	if err != nil {
		log.Fatal(err)
	}
	return path
}

func walletExists(db *bolt.DB) bool {
	exists := false
	db.View(func(tx *bolt.Tx) error {
		walletMetadata := tx.Bucket([]byte(walletMetadataBucket))
		utxosb := tx.Bucket([]byte(utxosBucket))
		keysb := tx.Bucket([]byte(keysBucket))
		authb := tx.Bucket([]byte(authBucket))

		if utxosb != nil && keysb != nil && authb != nil && walletMetadata != nil {
			if walletMetadata.Get([]byte(masterSeedKey)) != nil {
				exists = true
			}
		}
		return nil
	})
	return exists
}

func LoadWallet(rpcuser, rpcpass string) (*Wallet, error) {
	path := setupWalletDir()
	db, err := bolt.Open(filepath.Join(path, "wallet.db"), 0600, nil)
	if err != nil {
		return nil, fmt.Errorf("error opening db: %v", err)
	}

	if !walletExists(db) {
		return nil, ErrWalletNotExists
	}

	// TODO: handle difference between bitcoin core and btcd
	certHomeDir := btcutil.AppDataDir("btcd", false)
	certs, err := os.ReadFile(filepath.Join(certHomeDir, "rpc.cert"))
	connCfg := &rpcclient.ConnConfig{
		Host:         "localhost:18556",
		User:         rpcuser,
		Pass:         rpcpass,
		Certificates: certs,
		HTTPPostMode: true,
		// DisableTLS:   true,
	}

	client, err := rpcclient.New(connCfg, nil)
	if err != nil {
		return nil, fmt.Errorf("error setting up rpc client: %v", err)
	}

	addresses := make(map[address]derivationPath)
	wallet := &Wallet{db: db, client: client, addresses: addresses}

	wallet.balance = wallet.getBalance()
	wallet.lastExternalIdx = wallet.getLastExternalIdx()
	wallet.lastInternalIdx = wallet.getLastInternalIdx()
	wallet.lastScannedBlock = wallet.getLastScannedBlock()

	err = wallet.loadAddresses()
	if err != nil {
		return nil, err
	}

	err = wallet.loadUTXOs()
	if err != nil {
		return nil, err
	}

	return wallet, nil
}
