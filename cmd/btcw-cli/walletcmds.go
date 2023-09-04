package main

import (
	"bufio"
	"errors"
	"fmt"
	"net/rpc/jsonrpc"
	"os"
	"strconv"
	"time"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/elnosh/btcw/rpcserver"
	"github.com/urfave/cli/v2"
	"golang.org/x/term"
)

const (
	maxWalletUnlockDuration = 3600 // one hour
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

	// TODO: check this is a valid address
	addr := cliArgs.Get(0)

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
		return err
	}

	fmt.Println(*reply)
	return nil
}

var walletPassphraseCmd = &cli.Command{
	Name:   "walletpassphrase",
	Action: walletPassphrase,
}

func walletPassphrase(ctx *cli.Context) error {
	client, err := jsonrpc.Dial("tcp", "localhost:18557")
	if err != nil {
		return err
	}

	fmt.Println("enter passphrase to unlock wallet: ")
	passphrase, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return errors.New("error reading passphrase, please try again")
	}

	fmt.Println("provide duration (in seconds) to unlock wallet")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	durationInput := scanner.Text()

	durationInt, err := strconv.Atoi(durationInput)
	if err != nil {
		return errors.New("invalid time provided. plese enter duration in seconds")
	}

	if durationInt > maxWalletUnlockDuration {
		return errors.New("unlock duration time too high. provide a duration below 3600 seconds (one hour)")
	}
	timeSeconds := time.Second * time.Duration(durationInt)

	args := rpcserver.WalletPassphraseArgs{
		Passphrase: string(passphrase),
		Duration:   timeSeconds,
	}
	var reply *string

	err = client.Call("WalletRPC.WalletPassphrase", args, &reply)
	if err != nil {
		return err
	}

	return nil
}
