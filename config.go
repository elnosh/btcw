package main

import (
	"flag"
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
