package main

import (
	"log"
	"os"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "btcw cli",
		Usage: "cli tool for btcw",
		Commands: []*cli.Command{
			getBalanceCmd,
			getNewAddressCmd,
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
