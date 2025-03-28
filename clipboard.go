package main

import (
	"context"
	"log"
	"sync"
	"time"

	"golang.design/x/clipboard"
)

type TimeStamp = int64

const (
	TypeText  = "text"
	TypeImage = "image"
)

var (
	ClipboardText         *string
	ClipboardImage        *[]byte
	ClipboardLatest       string    = "" // text or image
	ClipboardLocalVersion TimeStamp = 0  // timestamp
	ClipboardMu           sync.Mutex
	clipboardWatching     bool = true // close clipboard watching when set
)

func setClipboardVersion(version TimeStamp) {
	if version == 0 {
		ClipboardLocalVersion = time.Now().Unix()
	} else {
		ClipboardLocalVersion = version
	}
}

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
		if !clipboardWatching {
			clipboardWatching = true
			ClipboardMu.Unlock()
			continue
		}
		setClipboardVersion(0)
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
		if !clipboardWatching {
			clipboardWatching = true
			ClipboardMu.Unlock()
			continue
		}
		setClipboardVersion(0)
		log.Println("Image data:", len(data))
		*ClipboardImage = data
		ClipboardLatest = TypeImage
		ClipboardMu.Unlock()
	}
}

func SetClipboardText(text string) {
	ClipboardMu.Lock()
	defer ClipboardMu.Unlock()
	clipboardWatching = false
	clipboard.Write(clipboard.FmtText, []byte(text))
}

func SetClipboardImage(data []byte) {
	ClipboardMu.Lock()
	defer ClipboardMu.Unlock()
	clipboardWatching = false
	clipboard.Write(clipboard.FmtImage, data)
}
