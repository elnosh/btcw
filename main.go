package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/elnosh/btcw/rpcserver"
	"github.com/elnosh/btcw/wallet"
)

func main() {
	flags := parseFlags()

	if flags.Create {
		err := wallet.CreateWallet()
		if err != nil {
			printErr(err)
		}
	} else {
		if flags.RPCUser == "" || flags.RPCPass == "" {
			printErr(errors.New("RPC username and password are required to start wallet"))
		}

		w, err := wallet.LoadWallet(flags.RPCUser, flags.RPCPass)
		if err != nil {
			if err == wallet.ErrWalletNotExists {
				printErr(errors.New("A wallet does not exist. Please create one first with -create"))
			}
			printErr(err)
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go func() {
			errChan := make(chan error)
			go wallet.ScanForNewBlocks(ctx, w, errChan)
			err = <-errChan
			if err != nil {
				fmt.Println(err)
			}
		}()

		err = rpcserver.StartRPCServer(w)
		if err != nil {
			printErr(err)
		}
	}
}

func printErr(msg error) {
	fmt.Println(msg.Error())
	os.Exit(0)
}
