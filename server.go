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
	ExecPath  string
	IsAutoRun string
	HostName  string
	LocalIP   string
	Port      string
	Password  string
}

type ResponseFile struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

type ResponseFiles []ResponseFile

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
			clientName = "unknown"
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

	router.GET("/", guideHandler)
	router.Use(clientName(), auth())
	router.GET("/get", getHandler)
	// router.POST("/set", setHandler)
	router.POST("/settings", settingsHandler)

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

	c.HTML(http.StatusOK, "guide", guideTemplateParams{
		ExecPath:  os.Args[0],
		IsAutoRun: isAutoRunText,
		HostName:  hostname,
		LocalIP:   GetLocalIP(),
		Port:      fmt.Sprintf("%d", Port),
		Password:  Password,
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
	go Restart(port, password)
}

func getHandler(c *gin.Context) {
	if ClipboardLatest == TypeText {
		log.Println("get clipboard text")
		c.JSON(http.StatusOK, gin.H{
			"type": TypeText,
			"data": *ClipboardText,
		})
		defer SendNotification(fmt.Sprintf("To [%s]", c.GetString("clientName")), *ClipboardText)
		return
	}

	if ClipboardLatest == TypeImage {
		responseFiles := make([]ResponseFile, 0, 1)
		responseFiles = append(responseFiles, ResponseFile{
			"clipboard.png",
			base64.StdEncoding.EncodeToString(*ClipboardImage),
		})

		c.JSON(http.StatusOK, gin.H{
			"type": TypeImage,
			"data": responseFiles,
		})
		defer SendNotification(fmt.Sprintf("To [%s]", c.GetString("clientName")), "[Image]")
		return
	}

	c.JSON(http.StatusBadRequest, gin.H{"error": "unknown content type"})
}
