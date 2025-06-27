//go:build ignore
package main

import (
	"github.com/zsais/go-gin-prometheus"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.New()

	/*	// Optional custom metrics list
		customMetrics := []*ginprometheus.Metric{
			&ginprometheus.Metric{
				ID:	"1234",				// optional string
				Name:	"test_metric",			// required string
				Description:	"Counter test metric",	// required string
				Type:	"counter",			// required string
			},
			&ginprometheus.Metric{
				ID:	"1235",				// Identifier
				Name:	"test_metric_2",		// Metric Name
				Description:	"Summary test metric",	// Help Description
				Type:	"summary", // type associated with prometheus collector
			},
			// Type Options:
			//	counter, counter_vec, gauge, gauge_vec,
			//	histogram, histogram_vec, summary, summary_vec
		}
		p := ginprometheus.NewPrometheus("gin", customMetrics)
	*/

	p := ginprometheus.NewWithConfig(ginprometheus.Config{
		Subsystem: "gin",
		CustomLabels: map[string]string{
			"custom_label": "custom_value",
		},
	})

	p.Use(r)
	r.GET("/", func(c *gin.Context) {
		c.JSON(200, "Hello world!")
	})

	if err := r.Run(":29090"); err != nil {
		panic(err)
	}
}
