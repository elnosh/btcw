package main

import (
	"fmt"
	"log"
	"net/rpc/jsonrpc"
	"os"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "btcw cli",
		Usage: "cli tool for btcw",
		Commands: []*cli.Command{
			getbalanceCmd,
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

var getbalanceCmd = &cli.Command{
	Name:   "getbalance",
	Action: getBalance,
}

func getBalance(ctx *cli.Context) error {
	client, err := jsonrpc.Dial("tcp", "localhost:18557")
	if err != nil {
		return err
	}

	var args struct{}
	var reply *int64

	err = client.Call("WalletRPC.GetBalance", args, &reply)
	if err != nil {
		return err
	}

	fmt.Printf("balance: %v sats\n", *reply)
	return nil
}
