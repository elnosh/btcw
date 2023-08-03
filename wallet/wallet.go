package wallet

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"golang.org/x/term"
)

var ErrPass = errors.New("error reading passphrase, please try again")

func CreateWalletPrompt() (string, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("do you want to create a new wallet? (y/n)")
	input, err := reader.ReadString('\n')
	if err != nil {
		log.Fatal("error reading input, please try again")
	}
	input = strings.ToLower(strings.TrimSpace(input))

	encodedHash := ""
	if input == "y" || input == "yes" {
		fmt.Print("enter passphrase for wallet: \n")
		passphrase, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			return "", ErrPass
		}
		fmt.Print("confirm passphrase: \n")
		confirmPassphrase, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			return "", ErrPass
		}
		if !bytes.Equal(passphrase, confirmPassphrase) {
			return "", errors.New("passphrases do not match, please try again")
		}

		encodedHash, err = hashPassphrase(confirmPassphrase)
		if err != nil {
			return "", err
		}
	} else {
		os.Exit(0)
	}

	return encodedHash, nil
}
