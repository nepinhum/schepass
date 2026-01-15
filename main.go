package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/nepinhum/schepass/internal/config"
	"github.com/nepinhum/schepass/internal/secureio"
	"github.com/nepinhum/schepass/internal/vault"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "init":
		cmdInit(os.Args[2:])
	case "add":
		cmdAdd(os.Args[2:])
	case "get":
		cmdGet(os.Args[2:])
	case "list":
		cmdList(os.Args[2:])
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "schepass <command> [args]\n\n")
	fmt.Fprintf(os.Stderr, "Commands:\n")
	fmt.Fprintf(os.Stderr, "  init   Initialize a new vault\n")
	fmt.Fprintf(os.Stderr, "  add    Add or update an entry\n")
	fmt.Fprintf(os.Stderr, "  get    Get an entry\n")
	fmt.Fprintf(os.Stderr, "  list   List entry names\n")
}

func cmdInit(args []string) {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	vaultPath := fs.String("vault", "", "Path to vault file")
	fs.Parse(args)

	path := resolveVaultPath(*vaultPath)
	if _, err := os.Stat(path); err == nil {
		fatalf("vault already exists: %s", path)
	}

	pass, err := secureio.PromptPasswordConfirm("New master password: ", "Confirm password: ")
	if err != nil {
		fatal(err)
	}

	v := vault.New()
	if err := vault.Save(path, pass, v); err != nil {
		fatal(err)
	}

	fmt.Printf("initialized vault: %s\n", path)
}

func cmdAdd(args []string) {
	fs := flag.NewFlagSet("add", flag.ExitOnError)
	vaultPath := fs.String("vault", "", "Path to vault file")
	username := fs.String("user", "", "Username")
	password := fs.String("pass", "", "Password")
	notes := fs.String("notes", "", "Notes")
	name, flagArgs, err := extractName(args)
	if err != nil {
		fatal(err)
	}
	fs.Parse(flagArgs)
	path := resolveVaultPath(*vaultPath)

	master, err := secureio.PromptPassword("Master password: ")
	if err != nil {
		fatal(err)
	}

	v, err := vault.Load(path, master)
	if err != nil {
		fatal(err)
	}

	entryPass := *password
	if entryPass == "" {
		entryPass, err = secureio.PromptPassword("Entry password: ")
		if err != nil {
			fatal(err)
		}
	}

	entry := v.Entries[name]
	if entry.Accounts == nil {
		entry.Accounts = make(map[string]vault.Account)
	}
	accountKey := accountKey(*username)
	entry.Accounts[accountKey] = vault.Account{
		Username: *username,
		Password: entryPass,
		Notes:    *notes,
	}
	v.Entries[name] = entry

	if err := vault.Save(path, master, v); err != nil {
		fatal(err)
	}

	fmt.Printf("saved: %s\n", name)
}

func cmdGet(args []string) {
	fs := flag.NewFlagSet("get", flag.ExitOnError)
	vaultPath := fs.String("vault", "", "Path to vault file")
	username := fs.String("user", "", "Username")
	name, flagArgs, err := extractName(args)
	if err != nil {
		fatal(err)
	}
	fs.Parse(flagArgs)
	path := resolveVaultPath(*vaultPath)

	master, err := secureio.PromptPassword("Master password: ")
	if err != nil {
		fatal(err)
	}

	v, err := vault.Load(path, master)
	if err != nil {
		fatal(err)
	}

	entry, ok := v.Entries[name]
	if !ok {
		fatalf("entry not found: %s", name)
	}

	accountKey, account, err := pickAccount(entry, name, *username)
	if err != nil {
		fatal(err)
	}
	fmt.Printf("name: %s\n", name)
	if account.Username != "" {
		fmt.Printf("user: %s\n", account.Username)
	} else if accountKey != "default" {
		fmt.Printf("user: %s\n", accountKey)
	}
	fmt.Printf("pass: %s\n", account.Password)
	if account.Notes != "" {
		fmt.Printf("notes: %s\n", account.Notes)
	}
}

func cmdList(args []string) {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	vaultPath := fs.String("vault", "", "Path to vault file")
	fs.Parse(args)

	path := resolveVaultPath(*vaultPath)

	master, err := secureio.PromptPassword("Master password: ")
	if err != nil {
		fatal(err)
	}

	v, err := vault.Load(path, master)
	if err != nil {
		fatal(err)
	}

	for name := range v.Entries {
		fmt.Println(name)
	}
}

func accountKey(username string) string {
	if strings.TrimSpace(username) == "" {
		return "default"
	}
	return strings.TrimSpace(username)
}

func pickAccount(entry vault.Entry, name, user string) (string, vault.Account, error) {
	if len(entry.Accounts) == 0 {
		return "", vault.Account{}, errors.New("entry has no accounts")
	}
	if user != "" {
		account, ok := entry.Accounts[user]
		if !ok {
			return "", vault.Account{}, fmt.Errorf("account not found: %s", user)
		}
		return user, account, nil
	}
	if len(entry.Accounts) == 1 {
		for key, account := range entry.Accounts {
			return key, account, nil
		}
	}
	return "", vault.Account{}, fmt.Errorf("multiple accounts found for %s; use --user", name)
}

func extractName(args []string) (string, []string, error) {
	var name string
	rest := make([]string, 0, len(args))
	for _, arg := range args {
		if name == "" && !strings.HasPrefix(arg, "-") {
			name = arg
			continue
		}
		rest = append(rest, arg)
	}
	if name == "" {
		return "", rest, errors.New("entry name required")
	}
	return name, rest, nil
}

func resolveVaultPath(flagPath string) string {
	if flagPath != "" {
		return flagPath
	}
	path, err := config.DefaultVaultPath()
	if err != nil {
		fatal(err)
	}
	return path
}

func fatal(err error) {
	if err == nil {
		return
	}
	fatalf("%s", err)
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}
