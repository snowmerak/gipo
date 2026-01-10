package main

import (
	"os"

	"golang.org/x/term"
)

func readPassword() ([]byte, error) {
	b, err := term.ReadPassword(int(os.Stdin.Fd()))
	// term.ReadPassword doesn't print newline; mimic user pressing enter
	return b, err
}
