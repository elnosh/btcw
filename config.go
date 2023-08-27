package main

import (
	"flag"
	"fmt"
)

type Flags struct {
	Create  bool
	Simnet  bool
	Regtest bool
	RPCUser string
	RPCPass string
	Node    string
}

func parseFlags() (*Flags, error) {
	flags := &Flags{}
	flag.BoolVar(&flags.Create, "create", false, "Create a new wallet")
	flag.BoolVar(&flags.Simnet, "simnet", false, "specify simnet")
	flag.BoolVar(&flags.Regtest, "regtest", false, "specify regtest")
	flag.StringVar(&flags.RPCUser, "rpcuser", "", "RPC username")
	flag.StringVar(&flags.RPCPass, "rpcpass", "", "RPC password")
	flag.StringVar(&flags.Node, "node", "btcd", "Node backing wallet (core or btcd)")
	flag.Parse()

	if flags.Node != "btcd" && flags.Node != "core" {
		return nil, fmt.Errorf("Invalid node type. Please provide 'btcd' or 'core'")
	}

	if flags.Node == "core" && flags.Simnet {
		return nil, fmt.Errorf("simnet is not available with core. For core please specify testnet or regtest")
	}

	return flags, nil
}
