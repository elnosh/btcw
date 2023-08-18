package main

import (
	"fmt"
	"net/rpc/jsonrpc"

	"github.com/btcsuite/btcd/btcutil"
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

	sats := btcutil.Amount(*reply)
	fmt.Printf("balance: %v BTC\n", sats.ToBTC())
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
