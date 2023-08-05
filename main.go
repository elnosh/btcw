package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

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

		reader := bufio.NewReader(os.Stdin)
		fmt.Println("do you want to create a new wallet? (y/n)")
		input, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal("error reading input, please try again")
		}

		input = strings.ToLower(strings.TrimSpace(input))
		var encodedHash string
		if input == "y" || input == "yes" {
			encodedHash, err = wallet.PromptPassphrase()
			if err != nil {
				printExit(err.Error())
			}
		} else {
			os.Exit(0)
		}

		err = wallet.InitAuthBucket(db, encodedHash)
		if err != nil {
			printExit(err.Error())
		}

		seed, err := hdkeychain.GenerateSeed(hdkeychain.RecommendedSeedLen)
		if err != nil {
			printExit(err.Error())
		}

		fmt.Println("Next will be the master seed. Write it down and store securely. Anyone with access to the seed has access to the funds.")
		fmt.Printf("seed: %x\n", seed)

		err = wallet.InitWalletMetadataBucket(db, seed, encodedHash)
		if err != nil {
			printExit(err.Error())
		}

		err = wallet.InitUTXOBucket(db)
		if err != nil {
			printExit(err.Error())
		}
	}

}

func printExit(msg string) {
	fmt.Println(msg)
	os.Exit(0)
}
