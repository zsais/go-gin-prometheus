package main

import (
	"github.com/mcuadros/go-gin-prometheus"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/gin-gonic/gin"
)

func main() {
	p := ginprometheus.NewPrometheus()

	r := gin.New()
	r.Use(p.HandlerFunc())

	r.GET("/metrics", gin.WrapH(prometheus.UninstrumentedHandler()))
	r.GET("/", func(c *gin.Context) {
		c.JSON(200, "Hello world!")
	})

	r.Run(":29090")
}
