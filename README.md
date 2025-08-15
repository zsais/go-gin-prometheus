# go-gin-prometheus
[![](https://godoc.org/github.com/zsais/go-gin-prometheus?status.svg)](https://godoc.org/github.com/zsais/go-gin-prometheus) [![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Gin Web Framework Prometheus metrics exporter

## Installation

`$ go get github.com/zsais/go-gin-prometheus`

## Usage

```go
package main

import (
	"github.com/gin-gonic/gin"
	"github.com/zsais/go-gin-prometheus"
)

func main() {
	r := gin.New()

	// NewWithConfig is the recommended way to initialize the middleware
	p := ginprometheus.NewWithConfig(ginprometheus.Config{
		Subsystem: "gin",
	})
	p.Use(r)

	r.GET("/", func(c *gin.Context) {
		c.JSON(200, "Hello world!")
	})

	r.Run(":29090")
}
```

See the [example.go file](https://github.com/zsais/go-gin-prometheus/blob/master/example/example.go)

## Custom Labels

It is possible to add custom labels to all metrics.

```go
package main

import (
    "github.com/gin-gonic/gin"
    "github.com/zsais/go-gin-prometheus"
)

func main() {
    r := gin.New()

    // NewWithConfig is the recommended way to initialize the middleware
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

    r.Run(":29090")
}
```

## Disabling Request Body Reading

By default, this middleware reads the entire request body to calculate the request size. This can be expensive for large request bodies. You can disable this behavior by setting the `DisableBodyReading` option to `true`. When disabled, the middleware will use the `ContentLength` header to determine the request size.

```go
package main

import (
    "github.com/gin-gonic/gin"
    "github.com/zsais/go-gin-prometheus"
)

func main() {
    r := gin.New()

    // NewWithConfig is the recommended way to initialize the middleware
    p := ginprometheus.NewWithConfig(ginprometheus.Config{
        Subsystem: "gin",
        DisableBodyReading: true,
    })
    p.Use(r)

    r.GET("/", func(c *gin.Context) {
        c.JSON(200, "Hello world!")
    })

    r.Run(":29090")
}
```

## Preserving a low cardinality for the request counter

The request counter (`requests_total`) has a `url` label which,
although desirable, can become problematic in cases where your
application uses templated routes expecting a great number of
variations, as Prometheus explicitly recommends against metrics having
high cardinality dimensions:

https://prometheus.io/docs/practices/naming/#labels

If you have for instance a `/customer/:name` templated route and you
don't want to generate a time series for every possible customer name,
you could supply this mapping function to the middleware:

```go
package main

import (
	"strings"
	"github.com/gin-gonic/gin"
	"github.com/zsais/go-gin-prometheus"
)

func main() {
	r := gin.New()

	// NewWithConfig is the recommended way to initialize the middleware
	p := ginprometheus.NewWithConfig(ginprometheus.Config{
		Subsystem: "gin",
	})

	p.ReqCntURLLabelMappingFn = func(c *gin.Context) string {
		url := c.Request.URL.Path
		for _, p := range c.Params {
			if p.Key == "name" {
				url = strings.Replace(url, p.Value, ":name", 1)
				break
			}
		}
		return url
	}

	p.Use(r)

	r.GET("/", func(c *gin.Context) {
		c.JSON(200, "Hello world!")
	})

	r.Run(":29090")
}
```

which would map `/customer/alice` and `/customer/bob` to their
template `/customer/:name`, and thus preserve a low cardinality for
our metrics.

### Note for Contributors

The default branch of this repository will soon be renamed from `master` to `main`. To update your local clone after this change has been made, you can use the following commands:

```bash
git fetch origin
git checkout main
git branch -u origin/main
git branch -d master
```
