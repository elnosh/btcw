package main

import (
	"fmt"
	"log"
	"os"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "btcw-cli",
		Usage: "cli tool for btcw",
		Commands: []*cli.Command{
			getBalanceCmd,
			getNewAddressCmd,
			sendToAddressCmd,
			walletPassphraseCmd,
			walletLockCmd,
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func printErr(msg error) {
	fmt.Println(msg.Error())
	os.Exit(0)
}
