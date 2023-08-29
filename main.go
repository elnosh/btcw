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
	flags, err := parseFlags()
	if err != nil {
		printErr(err)
	}

	net := getNetwork(flags)
	if flags.Create {
		err := wallet.CreateWallet(net)
		if err != nil {
			printErr(err)
		}
	} else {
		if flags.RPCUser == "" || flags.RPCPass == "" {
			printErr(errors.New("RPC username and password are required to start wallet"))
		}

		w, err := wallet.LoadWallet(net, flags.RPCUser, flags.RPCPass, flags.Node)
		if err != nil {
			if err == wallet.ErrWalletNotExists {
				printErr(errors.New("A wallet does not exist. Please create one first with -create"))
			}
			printErr(fmt.Errorf("error loading wallet: %v", err))
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go func() {
			errChan := make(chan error)
			go wallet.ScanForNewBlocks(ctx, w, errChan)
			err = <-errChan
			if err != nil {
				w.LogError(err.Error())
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
