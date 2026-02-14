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

	vaultView := buildVaultView(state)
	addView := buildAddView(state)
	getView := buildGetView(state)
	listView := buildListView(state)
	changeView := buildPasswdView(state)
	content := container.NewStack(buildVaultView(state))
	content.Objects = []fyne.CanvasObject{vaultView}

	title := canvas.NewText("Schepass", a.Settings().Theme().Color(theme.ColorNameForeground, a.Settings().ThemeVariant()))
	title.TextSize = 32

	nav := buildNav(func(name string) {
		switch name {
		case "Vault":
			content.Objects = []fyne.CanvasObject{vaultView}
		case "Add Entry":
			content.Objects = []fyne.CanvasObject{addView}
		case "Get Entry":
			content.Objects = []fyne.CanvasObject{getView}
		case "List":
			content.Objects = []fyne.CanvasObject{listView}
		case "Passwd":
			content.Objects = []fyne.CanvasObject{changeView}
		default:
			content.Objects = []fyne.CanvasObject{vaultView}
		}
		content.Refresh()
	})

	sidebar := container.NewBorder(title, nil, nil, nil, nav)
	sidebarWrap := container.New(newFixedSizeLayoutExpand(fyne.NewSize(220, 0)), sidebar)
	split := container.NewBorder(nil, nil, sidebarWrap, nil, content)
	w.SetContent(split)
	w.Resize(fyne.NewSize(780, 520))
	w.ShowAndRun()
}

func buildNav(onSelect func(name string)) fyne.CanvasObject {
	items := []string{"Vault", "Add Entry", "Get Entry", "List", "Passwd"}
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

	var unlockBtn *widget.Button
	unlockBtn = widget.NewButton("Unlock", func() {
		action(status, unlockBtn, func() (string, func(), error) {
			state.vaultPath = normalizePath(pathEntry.Text)
			v, err := vault.Load(state.vaultPath, masterEntry.Text)
			if err != nil {
				return "", nil, err
			}
			state.vault = v
			state.master = masterEntry.Text
			return "vault unlocked", nil, nil
		})
	})

	newPass := widget.NewPasswordEntry()
	newPass.SetPlaceHolder("New master password")
	confirm := widget.NewPasswordEntry()
	confirm.SetPlaceHolder("Confirm password")

	var initBtn *widget.Button
	initBtn = widget.NewButton("Init Vault", func() {
		action(status, initBtn, func() (string, func(), error) {
			state.vaultPath = normalizePath(pathEntry.Text)
			if _, err := os.Stat(state.vaultPath); err == nil {
				return "", nil, errors.New("vault already exists")
			} else if err != nil && !os.IsNotExist(err) {
				return "", nil, err
			}
			if newPass.Text == "" {
				return "", nil, errors.New("password required")
			}
			if newPass.Text != confirm.Text {
				return "", nil, errors.New("passwords do not match")
			}
			v := vault.New()
			if err := vault.Save(state.vaultPath, newPass.Text, v); err != nil {
				return "", nil, err
			}
			state.vault = v
			state.master = newPass.Text
			return "vault initialized", nil, nil
		})
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

	var saveBtn *widget.Button
	saveBtn = widget.NewButton("Save", func() {
		action(status, saveBtn, func() (string, func(), error) {
			name := strings.TrimSpace(nameEntry.Text)
			if name == "" {
				return "", nil, errors.New("entry name required")
			}
			v, master, err := ensureVault(state, masterEntry.Text)
			if err != nil {
				return "", nil, err
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
				return "", nil, err
			}
			return "saved", nil, nil
		})
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

	var getBtn *widget.Button
	getBtn = widget.NewButton("Get", func() {
		action(status, getBtn, func() (string, func(), error) {
			name := strings.TrimSpace(nameEntry.Text)
			if name == "" {
				return "", nil, errors.New("entry name required")
			}
			v, _, err := ensureVault(state, masterEntry.Text)
			if err != nil {
				return "", nil, err
			}
			entry, ok := v.Entries[name]
			if !ok {
				return "", nil, errors.New("entry not found")
			}
			accountKey, account, err := pickAccount(entry, name, strings.TrimSpace(userEntry.Text))
			if err != nil {
				return "", nil, err
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
			outText := strings.Join(lines, "\n")
			return "ok", func() {
				output.SetText(outText)
			}, nil
		})
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

	var refresh *widget.Button
	refresh = widget.NewButton("Refresh", func() {
		action(status, refresh, func() (string, func(), error) {
			v, _, err := ensureVault(state, masterEntry.Text)
			if err != nil {
				return "", nil, err
			}
			names := make([]string, 0, len(v.Entries))
			for name := range v.Entries {
				names = append(names, name)
			}
			return fmt.Sprintf("%d entries", len(names)), func() {
				list.Length = func() int { return len(names) }
				list.UpdateItem = func(id int, obj fyne.CanvasObject) {
					obj.(*widget.Label).SetText(names[id])
				}
				list.Refresh()
			}, nil
		})
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

	var changeBtn *widget.Button
	changeBtn = widget.NewButton("Change", func() {
		action(status, changeBtn, func() (string, func(), error) {
			if next.Text == "" {
				return "", nil, errors.New("password required")
			}
			if next.Text != confirm.Text {
				return "", nil, errors.New("passwords do not match")
			}
			path := normalizePath(state.vaultPath)
			v, err := vault.Load(path, current.Text)
			if err != nil {
				return "", nil, err
			}
			if err := vault.Save(path, next.Text, v); err != nil {
				return "", nil, err
			}
			state.vault = v
			state.master = next.Text
			return "master password updated", nil, nil
		})
	})

	form := widget.NewForm(
		widget.NewFormItem("Current", current),
		widget.NewFormItem("New", next),
		widget.NewFormItem("Confirm", confirm),
	)
	return container.NewBorder(nil, status, nil, nil, container.NewVBox(form, changeBtn))
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

type fixedSizeLayoutExpand struct {
	size fyne.Size
}

func newFixedSizeLayoutExpand(size fyne.Size) fyne.Layout {
	return &fixedSizeLayoutExpand{size: size}
}

func (l *fixedSizeLayoutExpand) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	width := l.size.Width
	height := l.size.Height
	if width <= 0 {
		width = size.Width
	}
	if height <= 0 {
		height = size.Height
	}
	for _, obj := range objects {
		obj.Move(fyne.NewPos(0, 0))
		obj.Resize(fyne.NewSize(width, height))
	}
}

func (l *fixedSizeLayoutExpand) MinSize(objects []fyne.CanvasObject) fyne.Size {
	return l.size
}

func action(status *widget.Label, button *widget.Button, action func() (string, func(), error)) {
	fyne.Do(func() {
		button.Disable()
		status.SetText("Working...")
	})
	go func() {
		message, uiUpdate, err := action()
		fyne.Do(func() {
			if err != nil {
				status.SetText(err.Error())
			} else if message != "" {
				status.SetText(message)
			} else {
				status.SetText("Done.")
			}
			if uiUpdate != nil {
				uiUpdate()
			}
			button.Enable()
		})
	}()
}
