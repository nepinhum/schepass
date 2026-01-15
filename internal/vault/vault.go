package vault

import (
	"encoding/json"
	"errors"
	"os"
	"strings"

	"github.com/nepinhum/schepass/internal/crypt"
	"github.com/nepinhum/schepass/internal/storage"
)

var errEmptyPassword = errors.New("password required")

type Account struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password"`
	Notes    string `json:"notes,omitempty"`
}

type Entry struct {
	Accounts map[string]Account `json:"accounts,omitempty"`
}

type entryJSON struct {
	Accounts map[string]Account `json:"accounts,omitempty"`
	Username string             `json:"username,omitempty"`
	Password string             `json:"password,omitempty"`
	Notes    string             `json:"notes,omitempty"`
}

func (e *Entry) UnmarshalJSON(data []byte) error {
	var payload entryJSON
	if err := json.Unmarshal(data, &payload); err != nil {
		return err
	}
	if payload.Accounts != nil {
		e.Accounts = payload.Accounts
		return nil
	}
	if payload.Password == "" && payload.Username == "" && payload.Notes == "" {
		e.Accounts = make(map[string]Account)
		return nil
	}
	key := normalizeAccountKey(payload.Username)
	e.Accounts = map[string]Account{
		key: {
			Username: payload.Username,
			Password: payload.Password,
			Notes:    payload.Notes,
		},
	}
	return nil
}

func (e Entry) MarshalJSON() ([]byte, error) {
	payload := entryJSON{
		Accounts: e.Accounts,
	}
	return json.Marshal(payload)
}

type Vault struct {
	Entries map[string]Entry `json:"entries"`
}

func New() *Vault {
	return &Vault{
		Entries: make(map[string]Entry),
	}
}

func Load(path, password string) (*Vault, error) {
	if password == "" {
		return nil, errEmptyPassword
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	plaintext, err := crypt.Decrypt(password, data)
	if err != nil {
		return nil, err
	}
	var v Vault
	if err := json.Unmarshal(plaintext, &v); err != nil {
		return nil, err
	}
	if v.Entries == nil {
		v.Entries = make(map[string]Entry)
	}
	return &v, nil
}

func Save(path, password string, v *Vault) error {
	if password == "" {
		return errEmptyPassword
	}
	if v == nil {
		v = New()
	}
	plaintext, err := json.Marshal(v)
	if err != nil {
		return err
	}
	payload, err := crypt.Encrypt(password, plaintext, crypt.DefaultParams())
	if err != nil {
		return err
	}
	return storage.WriteFileAtomic(path, payload, 0o600)
}

func normalizeAccountKey(username string) string {
	key := strings.TrimSpace(username)
	if key == "" {
		return "default"
	}
	return key
}
