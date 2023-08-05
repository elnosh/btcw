package wallet

import (
	"bytes"
	"errors"
	"fmt"
	"os"

	"golang.org/x/term"
)

var ErrPass = errors.New("error reading passphrase, please try again")

func PromptPassphrase() (string, error) {
	encodedHash := ""
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

	return encodedHash, nil
}
