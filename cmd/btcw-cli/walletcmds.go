package main

import (
	"errors"
	"fmt"
	"net/rpc/jsonrpc"
	"strconv"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/elnosh/btcw/rpcserver"
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
		return fmt.Errorf("error generating address: %s", err.Error())
	}

	fmt.Printf("address: %v\n", *reply)
	return nil
}

var sendToAddressCmd = &cli.Command{
	Name:   "sendtoaddress",
	Action: SendToAddress,
}

func SendToAddress(ctx *cli.Context) error {
	client, err := jsonrpc.Dial("tcp", "localhost:18557")
	if err != nil {
		return err
	}

	cliArgs := ctx.Args()
	if cliArgs.Len() != 2 {
		return errors.New("please provide address and amount to send")
	}

	// check this is a valid address
	addr := cliArgs.Get(0)

	// check this is a valid amount and convert to float64
	amountStr := cliArgs.Get(1)

	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		return errors.New("invalid amount")
	}

	args := rpcserver.SendToArgs{
		Address: addr,
		Amount:  amount,
	}

	var reply *string

	err = client.Call("WalletRPC.SendToAddress", args, &reply)
	if err != nil {
		return fmt.Errorf("error sending amount: %s", err.Error())
	}

	fmt.Println(*reply)
	return nil
}
