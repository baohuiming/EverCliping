package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/getlantern/systray"
	"golang.org/x/sync/errgroup"
)

var Port int
var Password string

func OnReady() {
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return StartMDNSServer(ctx, Port)
	})

	g.Go(func() error {
		return StartHTTPServer(ctx, Port)
	})

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

	go func() {
		if err := g.Wait(); err != nil {
			log.Printf("Error: %v\n", err)
		}
		cancel()
		systray.Quit()
	}()
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
