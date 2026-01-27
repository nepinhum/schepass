# schepass

(Actually personal) password manager CLI with a local encrypted vault.

## CLI

### Commands

| Command | Description |
| --- | --- |
| `init` | Initialize a new vault |
| `add` | Add or update an entry |
| `get` | Get an entry |
| `list` | List entry names |
| `passwd` | Change master password |

### Flags

| Flag | Applies To | Description |
| --- | --- | --- |
| `--vault` | `init`, `add`, `get`, `list` | Path to vault file |
| `--user` | `add`, `get` | Username (account key) |
| `--pass` | `add` | Password (omit to prompt) |
| `--notes` | `add` | Notes |

### Examples

```sh
# Initialize vault (defaults to ~/.schepass/vault.bin).
./schepass init

# Add multiple accounts under one entry.
./schepass add github --user me
./schepass add github --user work --notes "company"

# Fetch a specific account.
./schepass get github --user work

# List entries.
./schepass list

# Change master password.
./schepass passwd
```

## Build

### Local build

```sh
go build ./...
```

### UI build

```sh
go build -o schepass-ui ./ui
```

## Install

### Linux / macOS

```sh
# Build first.
go build -o schepass .

# System-wide (may require sudo).
./install.sh

# User-local (no sudo).
./install.sh --user
```

### Windows (PowerShell)

```powershell
# Build first.
go build -o schepass.exe .

# Install to user-local bin.
.\install.ps1 -AddToPath
```

### Cross-compile (examples)

```sh
GOOS=linux GOARCH=amd64 go build -o schepass-linux-amd64 .
GOOS=linux GOARCH=arm64 go build -o schepass-linux-arm64 .
GOOS=darwin GOARCH=arm64 go build -o schepass-darwin-arm64 .
GOOS=windows GOARCH=amd64 go build -o schepass-windows-amd64.exe .
```

## Notes

- Vault location defaults to `~/.schepass/vault.bin` unless `--vault` is provided.
- Each entry can contain multiple accounts (use `--user` to select one).

## TODO

- Add clipboard copy with timeout
- [x] Add user-friendly GUI
