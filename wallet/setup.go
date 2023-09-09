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

	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/elnosh/btcw/utils"
	bolt "go.etcd.io/bbolt"
	"golang.org/x/term"
)

var (
	ErrPass            = errors.New("error reading passphrase, please try again")
	ErrWalletNotExists = errors.New("wallet does not exist")
)

func CreateWallet(net *chaincfg.Params) error {
	path := setupWalletDir(net)
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

	seed, err := hdkeychain.GenerateSeed(hdkeychain.RecommendedSeedLen)
	if err != nil {
		return fmt.Errorf("error creating wallet: %v", err)
	}

	fmt.Println("Next will be the master seed. Write it down and store securely. Anyone with access to the seed has access to the funds.")
	fmt.Printf("seed: %x\n", seed)

	if err = wallet.initWalletBuckets(seed, encodedHash, net); err != nil {
		return fmt.Errorf("error creating wallet: %v", err)
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

	encodedHash, err := utils.HashPassphrase(confirmPassphrase)
	if err != nil {
		return "", err
	}

	return encodedHash, nil
}

func setupWalletDir(net *chaincfg.Params) string {
	homedir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	path := filepath.Join(homedir, ".btcw", net.Name, "wallet")
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

func LoadWallet(net *chaincfg.Params, rpcuser, rpcpass, node string) (*Wallet, error) {
	path := setupWalletDir(net)
	db, err := bolt.Open(filepath.Join(path, "wallet.db"), 0600, nil)
	if err != nil {
		return nil, fmt.Errorf("error opening db: %v", err)
	}

	if !walletExists(db) {
		return nil, ErrWalletNotExists
	}

	wallet := NewWallet(db, net)
	wallet.balance = wallet.getBalance()
	wallet.lastExternalIdx = wallet.getLastExternalIdx()
	wallet.lastInternalIdx = wallet.getLastInternalIdx()
	wallet.lastScannedBlock = wallet.getLastScannedBlock()
	wallet.locked = true

	err = wallet.loadExternalAddresses()
	if err != nil {
		return nil, err
	}

	err = wallet.loadUTXOs()
	if err != nil {
		return nil, err
	}

	var client NodeClient
	switch node {
	case "btcd":
		client, err = SetupBtcdClient(wallet, net, rpcuser, rpcpass)
		if err != nil {
			return nil, fmt.Errorf("wallet.NewBtcdClient: %w", err)
		}
	case "core":
		client, err = SetupBitcoinCoreClient(wallet, net, rpcuser, rpcpass)
		if err != nil {
			return nil, fmt.Errorf("wallet.NewBitcoinCoreClient: %w", err)
		}
	default:
		return nil, fmt.Errorf("invalid node type")
	}
	wallet.client = client

	err = wallet.loadTxFilter()
	if err != nil {
		return nil, err
	}

	// if this is new wallet, set last scanned block to current height of chain - 10
	// no need to scan entire blockchain if wallet is new
	if wallet.lastScannedBlock == 0 {
		chainHeight, err := wallet.client.GetBlockCount()
		if err != nil {
			return nil, err
		}
		wallet.setLastScannedBlock(chainHeight - 10)
	}
	return wallet, nil
}
