package ginprometheus

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
)

type Prometheus struct {
	reqCnt               *prometheus.CounterVec
	reqDur, reqSz, resSz prometheus.Summary
}

func NewPrometheus() *Prometheus {
	p := &Prometheus{}

	p.reqCnt = prometheus.MustRegisterOrGet(prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "How many HTTP requests processed, partitioned by status code and HTTP method.",
		},
		[]string{"code", "method", "handler"},
	)).(*prometheus.CounterVec)

	p.reqDur = prometheus.MustRegisterOrGet(prometheus.NewSummary(
		prometheus.SummaryOpts{
			Subsystem: "http",
			Name:      "request_duration_microseconds",
			Help:      "The HTTP request latencies in microseconds.",
		},
	)).(prometheus.Summary)

	p.reqSz = prometheus.MustRegisterOrGet(prometheus.NewSummary(
		prometheus.SummaryOpts{
			Subsystem: "http",
			Name:      "request_size_bytes",
			Help:      "The HTTP request sizes in bytes.",
		},
	)).(prometheus.Summary)

	p.resSz = prometheus.MustRegisterOrGet(prometheus.NewSummary(
		prometheus.SummaryOpts{
			Subsystem: "http",
			Name:      "response_size_bytes",
			Help:      "The HTTP response sizes in bytes.",
		},
	)).(prometheus.Summary)

	return p
}

func (p *Prometheus) HandlerFunc() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		status := strconv.Itoa(c.Writer.Status())
		elapsed := float64(time.Since(start)) / float64(time.Microsecond)

		p.reqDur.Observe(elapsed)
		p.reqCnt.WithLabelValues(status, c.Request.Method, c.HandlerName()).Inc()
	}
}
