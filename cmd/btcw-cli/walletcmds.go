package main

import (
	"fmt"
	"net/rpc/jsonrpc"

	"github.com/urfave/cli/v2"
)

var getBalanceCmd = &cli.Command{
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

var getNewAddressCmd = &cli.Command{
	Name:   "getnewaddress",
	Action: getNewAddress,
}

func getNewAddress(ctx *cli.Context) error {
	client, err := jsonrpc.Dial("tcp", "localhost:18557")
	if err != nil {
		return err
	}

	var args struct{}
	var reply *string

	err = client.Call("WalletRPC.GetNewAddress", args, &reply)
	if err != nil {
		return fmt.Errorf("error generating address: %v", err)
	}

	fmt.Printf("address: %v\n", *reply)
	return nil
}
