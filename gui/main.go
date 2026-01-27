package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/nepinhum/schepass/internal/config"
	"github.com/nepinhum/schepass/internal/vault"
)

type appState struct {
	vaultPath string
	master    string
	vault     *vault.Vault
}

func main() {
	a := app.NewWithID("io.schepass.gui")
	a.Settings().SetTheme(&forcedVariant{Theme: theme.DefaultTheme(), variant: theme.VariantDark})
	w := a.NewWindow("Schepass")
	w.SetMaster()

	state := &appState{
		vaultPath: defaultVaultPath(),
	}

	content := container.NewStack(buildVaultView(state))

	title := canvas.NewText("Schepass", a.Settings().Theme().Color(theme.ColorNameForeground, a.Settings().ThemeVariant()))
	title.TextSize = 24

	nav := buildNav(func(name string) {
		switch name {
		case "Vault":
			content.Objects = []fyne.CanvasObject{buildVaultView(state)}
		case "Add Entry":
			content.Objects = []fyne.CanvasObject{buildAddView(state)}
		case "Get Entry":
			content.Objects = []fyne.CanvasObject{buildGetView(state)}
		case "List Entries":
			content.Objects = []fyne.CanvasObject{buildListView(state)}
		case "Change Master":
			content.Objects = []fyne.CanvasObject{buildPasswdView(state)}
		default:
			content.Objects = []fyne.CanvasObject{buildVaultView(state)}
		}
		content.Refresh()
	})

	sidebar := container.NewBorder(title, nil, nil, nil, nav)
	split := container.NewBorder(nil, nil, sidebar, nil, content)
	w.SetContent(split)
	w.Resize(fyne.NewSize(760, 520))
	w.ShowAndRun()
}

func buildNav(onSelect func(name string)) fyne.CanvasObject {
	items := []string{"Vault", "Add Entry", "Get Entry", "List Entries", "Change Master"}
	tree := &widget.Tree{
		ChildUIDs: func(uid string) []string {
			if uid == "" {
				return items
			}
			return []string{}
		},
		IsBranch: func(uid string) bool {
			return uid == ""
		},
		CreateNode: func(branch bool) fyne.CanvasObject {
			return widget.NewLabel("")
		},
		UpdateNode: func(uid string, branch bool, obj fyne.CanvasObject) {
			obj.(*widget.Label).SetText(uid)
		},
		OnSelected: func(uid string) {
			if uid != "" {
				onSelect(uid)
			}
		},
	}
	tree.Select("Vault")
	return tree
}

func buildVaultView(state *appState) fyne.CanvasObject {
	status := widget.NewLabel("")
	pathEntry := widget.NewEntry()
	pathEntry.SetText(state.vaultPath)

	masterEntry := widget.NewPasswordEntry()
	masterEntry.SetPlaceHolder("Master password")

	unlockBtn := widget.NewButton("Unlock", func() {
		state.vaultPath = normalizePath(pathEntry.Text)
		v, err := vault.Load(state.vaultPath, masterEntry.Text)
		if err != nil {
			status.SetText(err.Error())
			return
		}
		state.vault = v
		state.master = masterEntry.Text
		status.SetText("vault unlocked")
	})

	newPass := widget.NewPasswordEntry()
	newPass.SetPlaceHolder("New master password")
	confirm := widget.NewPasswordEntry()
	confirm.SetPlaceHolder("Confirm password")

	initBtn := widget.NewButton("Init Vault", func() {
		state.vaultPath = normalizePath(pathEntry.Text)
		if _, err := os.Stat(state.vaultPath); err == nil {
			status.SetText("vault already exists")
			return
		} else if err != nil && !os.IsNotExist(err) {
			status.SetText(err.Error())
			return
		}
		if newPass.Text == "" {
			status.SetText("password required")
			return
		}
		if newPass.Text != confirm.Text {
			status.SetText("passwords do not match")
			return
		}
		v := vault.New()
		if err := vault.Save(state.vaultPath, newPass.Text, v); err != nil {
			status.SetText(err.Error())
			return
		}
		state.vault = v
		state.master = newPass.Text
		status.SetText("vault initialized")
	})

	form := widget.NewForm(
		widget.NewFormItem("Vault Path", pathEntry),
		widget.NewFormItem("Master", masterEntry),
	)
	form2 := widget.NewForm(
		widget.NewFormItem("New Master", newPass),
		widget.NewFormItem("Confirm", confirm),
	)

	return container.NewBorder(nil, status, nil, nil,
		container.NewVBox(
			form,
			container.NewHBox(unlockBtn),
			widget.NewSeparator(),
			form2,
			container.NewHBox(initBtn),
		),
	)
}

func buildAddView(state *appState) fyne.CanvasObject {
	status := widget.NewLabel("")
	nameEntry := widget.NewEntry()
	userEntry := widget.NewEntry()
	passEntry := widget.NewPasswordEntry()
	notesEntry := widget.NewMultiLineEntry()
	masterEntry := widget.NewPasswordEntry()
	masterEntry.SetPlaceHolder("Master password (if not unlocked)")

	saveBtn := widget.NewButton("Save", func() {
		name := strings.TrimSpace(nameEntry.Text)
		if name == "" {
			status.SetText("entry name required")
			return
		}
		v, master, err := ensureVault(state, masterEntry.Text)
		if err != nil {
			status.SetText(err.Error())
			return
		}
		entry := v.Entries[name]
		if entry.Accounts == nil {
			entry.Accounts = make(map[string]vault.Account)
		}
		accountKey := accountKey(userEntry.Text)
		entry.Accounts[accountKey] = vault.Account{
			Username: userEntry.Text,
			Password: passEntry.Text,
			Notes:    notesEntry.Text,
		}
		v.Entries[name] = entry
		if err := vault.Save(state.vaultPath, master, v); err != nil {
			status.SetText(err.Error())
			return
		}
		status.SetText("saved")
	})

	form := widget.NewForm(
		widget.NewFormItem("Name", nameEntry),
		widget.NewFormItem("User", userEntry),
		widget.NewFormItem("Password", passEntry),
		widget.NewFormItem("Notes", notesEntry),
		widget.NewFormItem("Master", masterEntry),
	)
	return container.NewBorder(nil, status, nil, nil, container.NewVBox(form, saveBtn))
}

func buildGetView(state *appState) fyne.CanvasObject {
	status := widget.NewLabel("")
	nameEntry := widget.NewEntry()
	userEntry := widget.NewEntry()
	masterEntry := widget.NewPasswordEntry()
	masterEntry.SetPlaceHolder("Master password (if not unlocked)")

	output := widget.NewMultiLineEntry()
	output.Disable()

	getBtn := widget.NewButton("Get", func() {
		name := strings.TrimSpace(nameEntry.Text)
		if name == "" {
			status.SetText("entry name required")
			return
		}
		v, _, err := ensureVault(state, masterEntry.Text)
		if err != nil {
			status.SetText(err.Error())
			return
		}
		entry, ok := v.Entries[name]
		if !ok {
			status.SetText("entry not found")
			return
		}
		accountKey, account, err := pickAccount(entry, name, strings.TrimSpace(userEntry.Text))
		if err != nil {
			status.SetText(err.Error())
			return
		}
		lines := []string{
			fmt.Sprintf("name: %s", name),
		}
		if account.Username != "" {
			lines = append(lines, fmt.Sprintf("user: %s", account.Username))
		} else if accountKey != "default" {
			lines = append(lines, fmt.Sprintf("user: %s", accountKey))
		}
		lines = append(lines, fmt.Sprintf("pass: %s", account.Password))
		if account.Notes != "" {
			lines = append(lines, fmt.Sprintf("notes: %s", account.Notes))
		}
		output.SetText(strings.Join(lines, "\n"))
		status.SetText("ok")
	})

	form := widget.NewForm(
		widget.NewFormItem("Name", nameEntry),
		widget.NewFormItem("User", userEntry),
		widget.NewFormItem("Master", masterEntry),
	)
	return container.NewBorder(nil, status, nil, nil, container.NewVBox(form, getBtn, output))
}

func buildListView(state *appState) fyne.CanvasObject {
	status := widget.NewLabel("")
	masterEntry := widget.NewPasswordEntry()
	masterEntry.SetPlaceHolder("Master password (if not unlocked)")
	list := widget.NewList(
		func() int { return 0 },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(id int, obj fyne.CanvasObject) {},
	)

	refresh := widget.NewButton("Refresh", func() {
		v, _, err := ensureVault(state, masterEntry.Text)
		if err != nil {
			status.SetText(err.Error())
			return
		}
		names := make([]string, 0, len(v.Entries))
		for name := range v.Entries {
			names = append(names, name)
		}
		list.Length = func() int { return len(names) }
		list.UpdateItem = func(id int, obj fyne.CanvasObject) {
			obj.(*widget.Label).SetText(names[id])
		}
		list.Refresh()
		status.SetText(fmt.Sprintf("%d entries", len(names)))
	})

	form := widget.NewForm(widget.NewFormItem("Master", masterEntry))
	return container.NewBorder(nil, status, nil, nil, container.NewVBox(form, refresh, list))
}

func buildPasswdView(state *appState) fyne.CanvasObject {
	status := widget.NewLabel("")
	current := widget.NewPasswordEntry()
	current.SetPlaceHolder("Current master password")
	next := widget.NewPasswordEntry()
	next.SetPlaceHolder("New master password")
	confirm := widget.NewPasswordEntry()
	confirm.SetPlaceHolder("Confirm password")

	change := widget.NewButton("Change", func() {
		if next.Text == "" {
			status.SetText("password required")
			return
		}
		if next.Text != confirm.Text {
			status.SetText("passwords do not match")
			return
		}
		path := normalizePath(state.vaultPath)
		v, err := vault.Load(path, current.Text)
		if err != nil {
			status.SetText(err.Error())
			return
		}
		if err := vault.Save(path, next.Text, v); err != nil {
			status.SetText(err.Error())
			return
		}
		state.vault = v
		state.master = next.Text
		status.SetText("master password updated")
	})

	form := widget.NewForm(
		widget.NewFormItem("Current", current),
		widget.NewFormItem("New", next),
		widget.NewFormItem("Confirm", confirm),
	)
	return container.NewBorder(nil, status, nil, nil, container.NewVBox(form, change))
}

func normalizePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return defaultVaultPath()
	}
	return path
}

func defaultVaultPath() string {
	path, err := config.DefaultVaultPath()
	if err != nil {
		return ""
	}
	return path
}

func ensureVault(state *appState, master string) (*vault.Vault, string, error) {
	if state.vault != nil && state.master != "" {
		return state.vault, state.master, nil
	}
	if strings.TrimSpace(master) == "" {
		return nil, "", errors.New("master password required")
	}
	path := normalizePath(state.vaultPath)
	v, err := vault.Load(path, master)
	if err != nil {
		return nil, "", err
	}
	state.vaultPath = path
	state.vault = v
	state.master = master
	return v, master, nil
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
	return "", vault.Account{}, fmt.Errorf("multiple accounts found for %s; use user field", name)
}
