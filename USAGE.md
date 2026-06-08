# USAGE

## Commands

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
# Initialize vault (defaults to ~/.schepass/vault.bin)
./schepass init

# Add multiple accounts under one entry
./schepass add github --user me
./schepass add github --user work --notes "company"

# Fetch a specific account
./schepass get github --user work

# List entries
./schepass list

# Change master password
./schepass passwd
```
