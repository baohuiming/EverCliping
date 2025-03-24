package main

import (
	"context"
	"log"

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
		text := string(data)
		log.Println("Text data:", text)
		*ClipboardText = text
		ClipboardLatest = TypeText
	}
}

func WatchImage(ctx context.Context) {
	ch := clipboard.Watch(ctx, clipboard.FmtImage)
	for data := range ch {
		log.Println("Image data:", len(data))
		*ClipboardImage = data
		ClipboardLatest = TypeImage
	}
}

func SetClipboardText(text string) {
	clipboard.Write(clipboard.FmtText, []byte(text))
}

func SetClipboardImage(data []byte) {
	clipboard.Write(clipboard.FmtImage, data)
}
