package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
)

type Flags struct {
	Create  bool
	RPCUser string
	RPCPass string
}

func parseFlags() *Flags {
	flags := &Flags{}
	flag.BoolVar(&flags.Create, "create", false, "Create a new wallet")
	flag.StringVar(&flags.RPCUser, "rpcuser", "", "RPC username")
	flag.StringVar(&flags.RPCPass, "rpcpass", "", "RPC password")
	flag.Parse()

	return flags
}

func setupWalletDir() string {
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
