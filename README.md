# schepass

(Actually personal) password manager CLI with a local encrypted vault.

## Build

### Local build

```sh
go build ./...
```

## Install

### Linux / macOS

```sh
# For system (may require sudo)
./install.sh

# For user
./install.sh --user
```

### Windows

```powershell
.\install.ps1 -AddToPath

```

## Notes

- Vault location defaults to `~/.schepass/vault.bin` unless `--vault` is provided.
- Each entry can contain multiple accounts (use `--user` to select one).
- [USAGE](https://github.com/nepinhum/schepass/blob/master/USAGE.md)
