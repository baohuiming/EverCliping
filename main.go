package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"sync"

	"github.com/getlantern/systray"
)

var (
	wg            sync.WaitGroup
	Port          int
	Password      string
	RemoteVersion string
)

func setupTray(cancel context.CancelFunc) {
	systray.SetIcon(IconData)
	systray.SetTitle("EverCliping")
	systray.SetTooltip("EverCliping")

	isAutoRun, err := QueryAutoRun()
	if err != nil {
		log.Println("[Warn] Unable to query AutoRun status:", err)
	}
	mOpen := systray.AddMenuItem("连接指南 (Guide)", "Guide")
	systray.AddSeparator()
	mAutorun := systray.AddMenuItemCheckbox("开机自启 (Autorun)", "Autorun", isAutoRun)
	mQuit := systray.AddMenuItem("退出 (Exit)", "Exit")

	go func() {
		for {
			select {
			case <-mOpen.ClickedCh:
				OpenBrowser(fmt.Sprintf("http://localhost:%d", Port))
			case <-mAutorun.ClickedCh:
				if mAutorun.Checked() {
					err := DisableAutoRun()
					if err != nil {
						log.Println("[Warn] Unable to disable AutoRun:", err)
						continue
					}
					mAutorun.Uncheck()
				} else {
					err := EnableAutoRun()
					if err != nil {
						log.Println("[Warn] Unable to enable AutoRun:", err)
						continue
					}
					mAutorun.Check()
				}
			case <-mQuit.ClickedCh:
				cancel()
				systray.Quit()
			}
		}
	}()
}

func OnReady() {
	ctx, cancel := context.WithCancel(context.Background())

	ClipboardInit()

	// TODO: mDNS support
	// go func() {
	// 	defer wg.Done()
	// 	wg.Add(1)
	// 	StartMDNSServer(ctx, Port)
	// }()

	go func() {
		defer wg.Done()
		wg.Add(1)
		StartHTTPServer(ctx, Port)
	}()

	go func() {
		defer wg.Done()
		wg.Add(1)
		WatchText(ctx)
	}()

	go func() {
		defer wg.Done()
		wg.Add(1)
		WatchImage(ctx)
	}()

	setupTray(cancel)
}

func OnExit() {
	log.Println("Exit...")
}

func main() {
	flag.IntVar(&Port, "port", 9273, "HTTP Server Port")
	flag.StringVar(&Password, "password", "", "Password")
	flag.Parse()

	systray.Run(OnReady, OnExit)
}
