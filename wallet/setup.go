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
	bolt "go.etcd.io/bbolt"
	"golang.org/x/term"
)

var ErrPass = errors.New("error reading passphrase, please try again")

func CreateWallet() error {
	path := SetupWalletDir()
	db, err := bolt.Open(filepath.Join(path, "wallet.db"), 0600, nil)
	if err != nil {
		return errors.New("error setting wallet")
	}

	// check if a wallet already exists
	if err = db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("wallet_metadata"))
		if b != nil {
			return errors.New("wallet already exists")
		}
		return nil
	}); err != nil {
		return err
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

func SetupWalletDir() string {
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
