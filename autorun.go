package main

import (
	"os"
	"strings"

	"github.com/baohuiming/autorun"
)

var AutoRunConfig = autorun.AutoRunConfig{
	AppName:        "EverCliping",
	ExecutablePath: strings.Join(os.Args, " "),
	CompanyName:    "evercliping",
}

func QueryAutoRun() (bool, error) {
	return autorun.QueryAutoRun(&AutoRunConfig)
}

func DisableAutoRun() error {
	return autorun.DisableAutoRun(&AutoRunConfig)
}

func EnableAutoRun() error {
	return autorun.EnableAutoRun(&AutoRunConfig)
}
