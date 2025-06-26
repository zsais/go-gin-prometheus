//go:build ignore

package main

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zsais/go-gin-prometheus"
)

func main() {
	r := gin.New()

	// Create a new Prometheus middleware
	p := ginprometheus.NewWithConfig(ginprometheus.Config{
		Subsystem: "gin",
	})

	// Set up the push gateway
	p.SetPushGateway("http://localhost:9091", "/metrics", 5*time.Second)
	p.SetPushGatewayJob("gin-advanced-example")

	// Use the middleware
	p.Use(r)

	r.GET("/", func(c *gin.Context) {
		c.JSON(200, "Hello world!")
	})

	r.Run(":29091")
}