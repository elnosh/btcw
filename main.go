package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/elnosh/btcw/rpcserver"
	"github.com/elnosh/btcw/wallet"
	bolt "go.etcd.io/bbolt"
)

func main() {
	flags := parseFlags()

	if flags.Create {
		err := wallet.CreateWallet()
		if err != nil {
			printErr(err)
		}
	}

	path := wallet.SetupWalletDir()
	db, err := bolt.Open(filepath.Join(path, "wallet.db"), 0600, nil)
	if err != nil {
		printErr(err)
	}

	wallet := wallet.NewWallet(db)

	err = rpcserver.StartRPCServer(wallet)
	if err != nil {
		printErr(err)
	}

}

func printErr(msg error) {
	fmt.Println(msg.Error())
	os.Exit(0)
}
