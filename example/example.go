package main

import (
	"github.com/mcuadros/go-gin-prometheus"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.New()

	p := ginprometheus.NewPrometheus("gin")
	p.Use(r)

	r.GET("/", func(c *gin.Context) {
		c.JSON(200, "Hello world!")
	})

	r.Run(":29090")
}
