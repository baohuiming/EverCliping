package main

import (
	"os"
	"strings"

	"golang.org/x/sys/windows/registry"
)

const REG_KEY = "EverCliping"

var REG_VALUE = strings.Join(os.Args, " ")

func openAutoRunKey(access uint32) (registry.Key, error) {
	autorunKey := `SOFTWARE\Microsoft\Windows\CurrentVersion\Run`
	key, err := registry.OpenKey(registry.CURRENT_USER, autorunKey, access)
	if err != nil {
		return 0, err
	}
	return key, nil
}

func QueryAutoRun() (bool, error) {
	key, err := openAutoRunKey(registry.QUERY_VALUE)
	if err != nil {
		return false, err
	}
	defer key.Close()
	val, _, err := key.GetStringValue(REG_KEY)
	if err != nil {
		if err == registry.ErrNotExist {
			return false, nil
		}
		return false, err
	}
	if val == REG_VALUE {
		return true, nil
	}
	return false, nil
}

func EnableAutoRun(value string) error {
	key, err := openAutoRunKey(registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()
	return key.SetStringValue(REG_KEY, value)
}

func DisableAutoRun() error {
	key, err := openAutoRunKey(registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()
	return key.DeleteValue(REG_KEY)
}
