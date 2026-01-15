package secureio

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

var errPasswordMismatch = errors.New("passwords do not match")

// PromptPassword reads a password from the terminal without echoing.
func PromptPassword(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	pass, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(pass)), nil
}

// PromptPasswordConfirm reads a password twice and ensures they match.
func PromptPasswordConfirm(prompt, confirm string) (string, error) {
	first, err := PromptPassword(prompt)
	if err != nil {
		return "", err
	}
	second, err := PromptPassword(confirm)
	if err != nil {
		return "", err
	}
	if first != second {
		return "", errPasswordMismatch
	}
	return first, nil
}
