package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type profileStore map[string]guiForm

func profileDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("หา config dir ไม่สำเร็จ: %w", err)
	}
	return filepath.Join(base, "mariadb-installer"), nil
}

func profilePath() (string, error) {
	dir, err := profileDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "profiles.json"), nil
}

func loadProfileStore() (profileStore, error) {
	path, err := profilePath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return profileStore{}, nil
		}
		return nil, fmt.Errorf("อ่าน profile store ไม่สำเร็จ: %w", err)
	}
	store := profileStore{}
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, fmt.Errorf("profile store ไม่ใช่ JSON ที่ถูกต้อง: %w", err)
	}
	if store == nil {
		store = profileStore{}
	}
	return store, nil
}

func saveProfileStore(store profileStore) error {
	path, err := profilePath()
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("สร้างโฟลเดอร์ profile ไม่สำเร็จ: %w", err)
	}
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return fmt.Errorf("แปลง profile store เป็น JSON ไม่สำเร็จ: %w", err)
	}
	return os.WriteFile(path, data, 0o600)
}

func saveProfile(name string, form guiForm) error {
	name = normalizeProfileName(name)
	if name == "" {
		return fmt.Errorf("ต้องระบุชื่อ profile")
	}
	form.Password = ""
	form.SudoPassword = ""
	store, err := loadProfileStore()
	if err != nil {
		return err
	}
	store[name] = form
	return saveProfileStore(store)
}

func deleteProfile(name string) error {
	name = normalizeProfileName(name)
	if name == "" {
		return fmt.Errorf("ต้องระบุชื่อ profile")
	}
	store, err := loadProfileStore()
	if err != nil {
		return err
	}
	delete(store, name)
	return saveProfileStore(store)
}

func loadProfile(name string) (guiForm, error) {
	name = normalizeProfileName(name)
	if name == "" {
		return guiForm{}, fmt.Errorf("ต้องระบุชื่อ profile")
	}
	store, err := loadProfileStore()
	if err != nil {
		return guiForm{}, err
	}
	form, ok := store[name]
	if !ok {
		return guiForm{}, fmt.Errorf("ไม่พบ profile %q", name)
	}
	if form.SSHPort == 0 {
		form.SSHPort = 22
	}
	if form.Mode == "" {
		form.Mode = "single"
	}
	if form.Action == "" {
		form.Action = "dry-run"
	}
	if form.Auth == "" {
		form.Auth = "password"
	}
	if form.Charset == "" {
		form.Charset = "utf8mb4"
	}
	return form, nil
}

func profileNames() ([]string, error) {
	store, err := loadProfileStore()
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(store))
	for name := range store {
		names = append(names, name)
	}
	sort.Strings(names)
	return names, nil
}

func normalizeProfileName(name string) string {
	return strings.TrimSpace(name)
}
