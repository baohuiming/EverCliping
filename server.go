package main

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/base64"
	"fmt"
	"html/template"
	"image/png"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/image/bmp"
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

func setupRouter() *gin.Engine {
	router := gin.Default()

	router.Use(clientName())

	tmpl := template.Must(template.New("guide").Parse(GuideTemplate))
	router.SetHTMLTemplate(tmpl)

	router.GET("/favicon.ico", func(c *gin.Context) {
		c.Data(http.StatusOK, "image/x-icon", IconData)
	})

	router.GET("/", guideHandler)
	router.GET("/get", getHandler)
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
		ExecPath:  EXEC_PATH,
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
	contentType, err := Clipboard().ContentType()
	if err != nil {
		log.Println("failed to get content type of clipboard")
		c.Status(http.StatusBadRequest)
		return
	}

	if contentType == TypeText {
		str, err := Clipboard().Text()
		if err != nil {
			c.Status(http.StatusBadRequest)
			log.Println("[Warn] failed to get clipboard")
			return
		}
		log.Println("get clipboard text")
		c.JSON(http.StatusOK, gin.H{
			"type": "text",
			"data": str,
		})
		defer SendNotification(fmt.Sprintf("To  [%s]", c.GetString("clientName")), str)
		return
	}

	if contentType == TypeBitmap {
		bmpBytes, err := Clipboard().Bitmap()
		if err != nil {
			log.Println("failed to get bmp bytes from clipboard")
		}

		bmpBytesReader := bytes.NewReader(bmpBytes)
		bmpImage, err := bmp.Decode(bmpBytesReader)
		if err != nil {
			log.Println("failed to decode bmp")
			c.JSON(http.StatusBadRequest, gin.H{"error": "unable to get clipboard content"})
			return
		}
		pngBytesBuffer := new(bytes.Buffer)
		if err = png.Encode(pngBytesBuffer, bmpImage); err != nil {
			log.Println("failed to encode bmp as png")
		}

		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "unable to get clipboard content"})
			return
		}

		responseFiles := make([]ResponseFile, 0, 1)
		responseFiles = append(responseFiles, ResponseFile{
			"clipboard.png",
			base64.StdEncoding.EncodeToString(pngBytesBuffer.Bytes()),
		})

		c.JSON(http.StatusOK, gin.H{
			"type": "file",
			"data": responseFiles,
		})
		defer SendNotification(fmt.Sprintf("To  [%s]", c.GetString("clientName")), "[Image]")
		return
	}

	if contentType == TypeFile {
		// get path of files from clipboard
		filenames, err := Clipboard().Files()
		if err != nil {
			log.Println("failed to get path of files from clipboard")
			c.Status(http.StatusBadRequest)
			return
		}

		responseFiles := make([]ResponseFile, 0, len(filenames))
		for _, path := range filenames {
			base64, err := ReadBase64FromFile(path)
			if err != nil {
				log.Println("read base64 from file failed")
				continue
			}
			responseFiles = append(responseFiles, ResponseFile{filepath.Base(path), base64})
		}
		log.Println("get clipboard files")

		c.JSON(http.StatusOK, gin.H{
			"type": "file",
			"data": responseFiles,
		})

		defer SendNotification(fmt.Sprintf("To  [%s]", c.GetString("clientName")), fmt.Sprintf("Files[%s]", strings.Join(filenames, ", ")))
		return
	}

	c.JSON(http.StatusBadRequest, gin.H{"error": "unknown content type"})
}

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
