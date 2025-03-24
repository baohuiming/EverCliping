package main

import (
	"context"
	"log"
	"sync"

	"golang.design/x/clipboard"
)

const (
	TypeText  = "text"
	TypeImage = "image"
)

var (
	ClipboardText   *string
	ClipboardImage  *[]byte
	ClipboardLatest string = ""
	ClipboardMu     sync.Mutex
)

func ClipboardInit() {
	err := clipboard.Init()
	if err != nil {
		panic(err)
	}

	ClipboardText = new(string)
	ClipboardImage = new([]byte)
}

func WatchText(ctx context.Context) {
	ch := clipboard.Watch(ctx, clipboard.FmtText)
	for data := range ch {
		ClipboardMu.Lock()
		text := string(data)
		log.Println("Text data:", text)
		*ClipboardText = text
		ClipboardLatest = TypeText
		ClipboardMu.Unlock()
	}
}

func WatchImage(ctx context.Context) {
	ch := clipboard.Watch(ctx, clipboard.FmtImage)
	for data := range ch {
		ClipboardMu.Lock()
		log.Println("Image data:", len(data))
		*ClipboardImage = data
		ClipboardLatest = TypeImage
		ClipboardMu.Unlock()
	}
}

func SetClipboardText(text string) {
	ClipboardMu.Lock()
	defer ClipboardMu.Unlock()
	clipboard.Write(clipboard.FmtText, []byte(text))
}

func SetClipboardImage(data []byte) {
	ClipboardMu.Lock()
	defer ClipboardMu.Unlock()
	clipboard.Write(clipboard.FmtImage, data)
}
