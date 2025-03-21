package main

import (
	"encoding/base64"
	"log"
	"net"
	"os"
	"os/exec"
	"runtime"
	"syscall"

	"github.com/gen2brain/beeep"
)

func GetLocalIP() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		log.Println("cannot get interfaces:", err)
		return ""
	}

	for _, iface := range interfaces {
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}

			ipv4 := ipNet.IP.To4()
			if ipv4 == nil || ipv4.IsLoopback() {
				continue
			}

			return ipv4.String()
		}
	}
	return ""
}

func OpenBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
	}
	args = append(args, url)

	return exec.Command(cmd, args...).Start()
}

func SendNotification(title, message string) {
	err := beeep.Notify(title, message, "")
	if err != nil {
		log.Println("failed to send notification:", err)
	}
}

func ReadBase64FromFile(path string) (string, error) {
	fileBytes, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(fileBytes), nil
}

func Restart(port, password string) {
	exePath := os.Args[0]
	args := []string{"-port", port, "-password", password}

	cmd := exec.Command(exePath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}

	if err := cmd.Start(); err != nil {
		panic(err)
	}
	os.Exit(0)
}
