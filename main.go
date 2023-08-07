package main

import (
	"fmt"
	"os"

	"github.com/elnosh/btcw/rpcserver"
	"github.com/elnosh/btcw/wallet"
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

	if flags.Create {
		err := wallet.CreateWallet()
		if err != nil {
			printErr(err)
		}
	}

	wallet := &wallet.Wallet{}

	err := rpcserver.StartRPCServer(wallet)
	if err != nil {
		printErr(err)
	}

}

func printErr(msg error) {
	fmt.Println(msg.Error())
	os.Exit(0)
}
