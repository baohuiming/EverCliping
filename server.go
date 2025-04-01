package main

import (
	"context"
	_ "embed"
	"encoding/base64"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type guideTemplateParams struct {
	ShortcutsURL     string
	ExecPath         string
	IsAutoRun        string
	HostName         string
	LocalIP          string
	Port             string
	Password         string
	Notify           string
	ConnectedDevices string
}

// Store the device's last poll timestamp
var DeviceStates = make(map[string]TimeStamp)
var DeviceAlive = 120 * time.Second

func StartHTTPServer(ctx context.Context, port int) error {
	// gin.SetMode(gin.ReleaseMode)
	router := setupRouter()

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: router,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("Server forced to shutdown: %v", err)
		}
	}()

	log.Printf("Server starting on http://localhost:%d", port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server failed to start: %v", err)
	}

	log.Printf("HTTP server shutting down.")
	return nil
}

func clientName() gin.HandlerFunc {
	return func(c *gin.Context) {
		urlEncodedClientName := c.GetHeader("X-Client-Name")
		clientName, err := url.PathUnescape(urlEncodedClientName)
		if err != nil || clientName == "" {
			clientName = "<Unknown Device>"
		}
		c.Set("clientName", clientName)
		c.Next()
	}
}

func auth() gin.HandlerFunc {
	return func(c *gin.Context) {

		if Password == "" {
			c.Next()
			return
		}

		reqAuth := c.GetHeader("X-Password")

		if Password == reqAuth {
			c.Next()
			return
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"error": "wrong password",
		})
	}
}

func setupRouter() *gin.Engine {
	router := gin.Default()

	tmpl := template.Must(template.New("guide").Parse(GuideTemplate))
	router.SetHTMLTemplate(tmpl)

	router.GET("/favicon.ico", func(c *gin.Context) {
		c.Data(http.StatusOK, "image/x-icon", IconData)
	})
	router.GET("/shortcuts.png", func(c *gin.Context) {
		c.Data(http.StatusOK, "image/png", ShortcutsData)
	})

	router.GET("/", guideHandler)
	router.Use(clientName(), auth())
	router.GET("/get", getHandler)
	router.POST("/set", setHandler)
	router.POST("/settings", settingsHandler)
	router.GET("/poll", pollHandler)

	return router
}

func guideHandler(c *gin.Context) {
	isAutoRun, err := QueryAutoRun()
	isAutoRunText := "否(No)"
	if err != nil {
		log.Println("[Warn] Unable to query AutoRun status:", err)
	} else if isAutoRun {
		isAutoRunText = "是(Yes)"
	}

	hostname, _ := os.Hostname()

	notify := ""
	if Notify != 0 {
		notify = "checked"
	}

	c.HTML(http.StatusOK, "guide", guideTemplateParams{
		ShortcutsURL: ShortcutsURL,
		ExecPath:     os.Args[0],
		IsAutoRun:    isAutoRunText,
		HostName:     hostname,
		LocalIP:      GetLocalIP(),
		Port:         fmt.Sprintf("%d", Port),
		Password:     Password,
		Notify:       notify,
		ConnectedDevices: func() string {
			var devices string
			for device, timestamp := range DeviceStates {
				devices += fmt.Sprintf("%s(%s), ", device, time.Unix(timestamp, 0).Format("2006/01/02 15:04:05"))
			}
			return devices
		}(),
	})
}

func settingsHandler(c *gin.Context) {
	port := c.PostForm("port")
	if port != "" {
		if portInt, err := strconv.Atoi(port); err != nil || portInt <= 0 || portInt > 65536 {
			log.Println("invalid port number")
			c.JSON(http.StatusBadRequest, gin.H{"error": "Port number must be between 1 and 65535"})
			return
		}
	} else {
		port = fmt.Sprintf("%d", Port)
	}
	password := c.PostForm("password")
	log.Println("[!]Notify:", c.PostForm("notify"))

	notify := 0
	if c.PostForm("notify") == "on" {
		notify = 1
	}

	go Restart(port, password, notify)
}

func getHandler(c *gin.Context) {
	if ClipboardLatest == TypeText {
		log.Println("get clipboard text")
		c.JSON(http.StatusOK, gin.H{
			"type":    TypeText,
			"data":    *ClipboardText,
			"version": ClipboardLocalVersion,
		})
		defer SendNotification(fmt.Sprintf("To [%s]", c.GetString("clientName")), *ClipboardText)
		return
	}

	if ClipboardLatest == TypeImage {
		c.JSON(http.StatusOK, gin.H{
			"type":    TypeImage,
			"data":    base64.StdEncoding.EncodeToString(*ClipboardImage),
			"version": ClipboardLocalVersion,
		})
		defer SendNotification(fmt.Sprintf("To [%s]", c.GetString("clientName")), "[Image]")
		return
	}

	c.JSON(http.StatusBadRequest, gin.H{"error": "unknown content type"})
}

type ReqBody struct {
	Data string `json:"data"`
}

func setHandler(c *gin.Context) {
	contentType := c.GetHeader("X-Content-Type")
	remoteVersion := c.GetHeader("X-Version")

	if remoteVersion == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing version"})
		return
	}

	version, err := strconv.ParseInt(remoteVersion, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid version"})
		return
	}

	setClipboardVersion(version)

	if contentType == TypeText {
		setTextHandler(c)
	} else if contentType == TypeImage {
		setImageHandler(c)
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown content type"})
		return
	}
}

func setTextHandler(c *gin.Context) {
	var body ReqBody
	if err := c.ShouldBindJSON(&body); err != nil {
		log.Println("failed to bind text body")
		c.Status(http.StatusBadRequest)
		return
	}

	SetClipboardText(body.Data)

	var notify string = "<empty>"
	if body.Data != "" {
		notify = body.Data
	}
	defer SendNotification(fmt.Sprintf("From [%s]", c.GetString("clientName")), notify)
	log.Println("set clipboard text")
	c.Status(http.StatusOK)
}

func setImageHandler(c *gin.Context) {
	var body ReqBody

	if err := c.ShouldBindJSON(&body); err != nil {
		log.Println("failed to bind image body")
		c.Status(http.StatusBadRequest)
		return
	}

	imgBytes, err := base64.StdEncoding.DecodeString(body.Data)
	if err != nil {
		log.Println("failed to decode base64 image")
		c.Status(http.StatusBadRequest)
		return
	}

	SetClipboardImage(imgBytes)

	defer SendNotification(fmt.Sprintf("From [%s]", c.GetString("clientName")), "[Image]")
	log.Println("set clipboard image")
	c.Status(http.StatusOK)
}

func pollHandler(c *gin.Context) {
	reqVersion := c.GetHeader("X-Version")
	client := c.GetString("clientName")

	defer func() {
		DeviceStates[client] = TimeStamp(time.Now().Unix())
	}()

	lastPoll, ok := DeviceStates[client]
	isFirstPoll := false
	if !ok || time.Since(time.Unix(lastPoll, 0)) > DeviceAlive { // new device
		isFirstPoll = true
	}

	if reqVersion == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing version"})
		return
	}

	remoteVersion, err := strconv.ParseInt(reqVersion, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid version"})
		return
	}

	if remoteVersion == ClipboardLocalVersion {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
		return
	} else if remoteVersion < ClipboardLocalVersion {
		// from PC to phone
		log.Println("from PC to phone: ", remoteVersion, "<", ClipboardLocalVersion)
		getHandler(c)
		return
	} else {
		// from phone to PC
		if isFirstPoll {
			setClipboardVersion(remoteVersion)
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
			return
		}
		log.Println("from phone to PC: ", remoteVersion, ">", ClipboardLocalVersion)

		c.JSON(http.StatusOK, gin.H{"status": "conflict"})
	}
}
