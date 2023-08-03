package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/elnosh/btcw/wallet"
	bolt "go.etcd.io/bbolt"
)

func main() {
	flags := parseFlags()

	// nodeConnCfg := &rpcclient.ConnConfig{
	// 	Host:         "localhost:18332",
	// 	User:         flags.RPCUser,
	// 	Pass:         flags.RPCPass,
	// 	HTTPPostMode: true,
	// 	DisableTLS:   true,
	// }

	// client, err := rpcclient.New(nodeConnCfg, nil)
	// if err != nil {
	// 	log.Fatal("error starting wallet")
	// }

	path := setupWalletDir()
	db, err := bolt.Open(filepath.Join(path, "wallet.db"), 0600, nil)
	if err != nil {
		log.Fatal("error setting wallet")
	}
	defer db.Close()

	if flags.Create {
		// check if a wallet already exists. if not, initiate prompt to create it
		db.View(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte("wallet_metadata"))
			if b != nil {
				printExit("wallet already exists")
			}
			return nil
		})

		encodedHash, err := wallet.CreateWalletPrompt()
		if err != nil {
			printExit(err.Error())
		}

		// set hashed passphrase in db
		if err = db.Update(func(tx *bolt.Tx) error {
			b, err := tx.CreateBucket([]byte("auth"))
			if err != nil {
				return err
			}
			return b.Put([]byte("encodedhash"), []byte(encodedHash))
		}); err != nil {
			printExit(err.Error())
		}

		// 2. generate master seed and have prompts to make sure user writes it down
		seed, err := hdkeychain.GenerateSeed(hdkeychain.RecommendedSeedLen)
		if err != nil {
			// handle err
		}

		fmt.Println("Next will be the master seed. Write it down and store securely. Anyone with access to the seed has access to the funds.")
		fmt.Printf("seed: %x\n", seed)

		master, acct0, err := wallet.DeriveHDKeys(seed, encodedHash)
		if err != nil {
			printExit(err.Error())
		}

		_, key, _, err := wallet.DecodeKey(encodedHash)
		if err != nil {
			printExit(err.Error())
		}

		encryptedMaster, err := wallet.Encrypt([]byte(master.String()), key)
		if err != nil {
			printExit(err.Error())
		}

		// external chain of account 0 - path: m/44'/1'/0'/0
		acct0ext, err := acct0.Derive(0)
		if err != nil {
			printExit(err.Error())
		}

		encryptedAcct0ext, err := wallet.Encrypt([]byte(acct0ext.String()), key)
		if err != nil {
			printExit(err.Error())
		}

		// internal chain of account 0 - path: m/44'/1'/0'/1
		acct0int, err := acct0.Derive(1)
		if err != nil {
			printExit(err.Error())
		}

		encryptedAcct0int, err := wallet.Encrypt([]byte(acct0int.String()), key)
		if err != nil {
			printExit(err.Error())
		}

		// create wallet metadata bucket
		if err = db.Update(func(tx *bolt.Tx) error {
			wallet, err := tx.CreateBucket([]byte("wallet_metadata"))
			if err != nil {
				return err
			}
			// set balance field key

			// set derivation paths needed
			if err = wallet.Put([]byte("master_seed"), encryptedMaster); err != nil {
				return err
			}

			if err = wallet.Put([]byte("account_0_external"), encryptedAcct0ext); err != nil {
				return err
			}
			if err = wallet.Put([]byte("account_0_internal"), encryptedAcct0int); err != nil {
				return err
			}
			return nil
		}); err != nil {
			// handle err
		}

	}
}

func printExit(msg string) {
	fmt.Println(msg)
	os.Exit(0)
}
