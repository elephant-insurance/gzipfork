package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/elephant-insurance/gzipfork"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	r.Use(gzipfork.Gzip(gzipfork.DefaultCompression))
	r.GET("/ping", func(c *gin.Context) {
		c.String(http.StatusOK, "pong "+fmt.Sprint(time.Now().Unix()))
	})

	// Listen and Server in 0.0.0.0:8080
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
