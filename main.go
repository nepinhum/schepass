package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/nepinhum/schepass/internal/config"
	"github.com/nepinhum/schepass/internal/message"
	"github.com/nepinhum/schepass/internal/secureio"
	"github.com/nepinhum/schepass/internal/vault"
)

func main() {
	// eh, something happened and I wanted to debug, and I will remove this!
	msgPath, err := config.DefaultMessagesPath()
	if err == nil {
		if err := message.LoadOrSeed(msgPath, "resources/default_messages.json"); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to load messages: %s\n", err)
		}
	}

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
	fmt.Fprint(os.Stderr, message.Msg("usage.header"))
	fmt.Fprint(os.Stderr, message.Msg("usage.commands_label"))
	fmt.Fprint(os.Stderr, message.Msg("usage.cmd_init"))
	fmt.Fprint(os.Stderr, message.Msg("usage.cmd_add"))
	fmt.Fprint(os.Stderr, message.Msg("usage.cmd_get"))
	fmt.Fprint(os.Stderr, message.Msg("usage.cmd_list"))
}

func cmdInit(args []string) {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	vaultPath := fs.String("vault", "", message.Msg("flags.vault"))
	fs.Parse(args)

	path := resolveVaultPath(*vaultPath)
	if _, err := os.Stat(path); err == nil {
		fatalf("%s", message.Msg("errors.vault_exists", path))
	}

	pass, err := secureio.PromptPasswordConfirm(
		message.Msg("prompt.new_master"),
		message.Msg("prompt.confirm_master"),
	)
	if err != nil {
		fatal(err)
	}

	v := vault.New()
	if err := vault.Save(path, pass, v); err != nil {
		fatal(err)
	}

	fmt.Printf(message.Msg("info.initialized"), path)
}

func cmdAdd(args []string) {
	fs := flag.NewFlagSet("add", flag.ExitOnError)
	vaultPath := fs.String("vault", "", message.Msg("flags.vault"))
	username := fs.String("user", "", message.Msg("flags.user"))
	password := fs.String("pass", "", message.Msg("flags.password"))
	notes := fs.String("notes", "", message.Msg("flags.notes"))
	name, flagArgs, err := extractName(args)
	if err != nil {
		fatal(err)
	}
	fs.Parse(flagArgs)
	path := resolveVaultPath(*vaultPath)

	master, err := secureio.PromptPassword(message.Msg("prompt.master"))
	if err != nil {
		fatal(err)
	}

	v, err := vault.Load(path, master)
	if err != nil {
		fatal(err)
	}

	entryPass := *password
	if entryPass == "" {
		entryPass, err = secureio.PromptPassword(message.Msg("prompt.entry_password"))
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

	fmt.Printf(message.Msg("info.saved"), name)
}

func cmdGet(args []string) {
	fs := flag.NewFlagSet("get", flag.ExitOnError)
	vaultPath := fs.String("vault", "", message.Msg("flags.vault"))
	username := fs.String("user", "", message.Msg("flags.user"))
	name, flagArgs, err := extractName(args)
	if err != nil {
		fatal(err)
	}
	fs.Parse(flagArgs)
	path := resolveVaultPath(*vaultPath)

	master, err := secureio.PromptPassword(message.Msg("prompt.master"))
	if err != nil {
		fatal(err)
	}

	v, err := vault.Load(path, master)
	if err != nil {
		fatal(err)
	}

	entry, ok := v.Entries[name]
	if !ok {
		fatalf("%s", message.Msg("errors.entry_not_found", name))
	}

	accountKey, account, err := pickAccount(entry, name, *username)
	if err != nil {
		fatal(err)
	}
	fmt.Printf(message.Msg("output.name"), name)
	if account.Username != "" {
		fmt.Printf(message.Msg("output.user"), account.Username)
	} else if accountKey != "default" {
		fmt.Printf(message.Msg("output.user"), accountKey)
	}
	fmt.Printf(message.Msg("output.pass"), account.Password)
	if account.Notes != "" {
		fmt.Printf(message.Msg("output.notes"), account.Notes)
	}
}

func cmdList(args []string) {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	vaultPath := fs.String("vault", "", message.Msg("flags.vault"))
	fs.Parse(args)

	path := resolveVaultPath(*vaultPath)

	master, err := secureio.PromptPassword(message.Msg("prompt.master"))
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
		return "", vault.Account{}, errors.New(message.Msg("errors.entry_no_accounts"))
	}
	if user != "" {
		account, ok := entry.Accounts[user]
		if !ok {
			return "", vault.Account{}, fmt.Errorf("%s", message.Msg("errors.account_not_found", user))
		}
		return user, account, nil
	}
	if len(entry.Accounts) == 1 {
		for key, account := range entry.Accounts {
			return key, account, nil
		}
	}
	return "", vault.Account{}, fmt.Errorf("%s", message.Msg("errors.multiple_accounts", name))
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
		return "", rest, errors.New(message.Msg("errors.entry_name_required"))
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
