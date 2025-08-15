package ginprometheus

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

var defaultMetricPath = "/metrics"

// Standard default metrics
//	counter, counter_vec, gauge, gauge_vec,
//	histogram, histogram_vec, summary, summary_vec
var reqCnt = &Metric{
	ID:          "reqCnt",
	Name:        "requests_total",
	Description: "How many HTTP requests processed, partitioned by status code and HTTP method.",
	Type:        "counter_vec",
	Args:        []string{"code", "method", "handler", "host", "url"}}

var reqDur = &Metric{
	ID:          "reqDur",
	Name:        "request_duration_seconds",
	Description: "The HTTP request latencies in seconds.",
	Type:        "histogram_vec",
	Args:        []string{"code", "method", "url"},
}

var resSz = &Metric{
	ID:          "resSz",
	Name:        "response_size_bytes",
	Description: "The HTTP response sizes in bytes.",
	Type:        "summary"}

var reqSz = &Metric{
	ID:          "reqSz",
	Name:        "request_size_bytes",
	Description: "The HTTP request sizes in bytes.",
	Type:        "summary"}

var standardMetrics = []*Metric{
	reqCnt,
	reqDur,
	resSz,
	reqSz,
}

/*
RequestCounterURLLabelMappingFn is a function which can be supplied to the middleware to control
the cardinality of the request counter's "url" label, which might be required in some contexts.
For instance, if for a "/customer/:name" route you don't want to generate a time series for every
possible customer name, you could use this function:

func(c *gin.Context) string {
	url := c.Request.URL.Path
	for _, p := range c.Params {
		if p.Key == "name" {
			url = strings.Replace(url, p.Value, ":name", 1)
			break
		}
	}
	return url
}

which would map "/customer/alice" and "/customer/bob" to their template "/customer/:name".
*/
type RequestCounterURLLabelMappingFn func(c *gin.Context) string

// Metric defines a prometheus metric. It is used to create a new prometheus
// collector.
type Metric struct {
	// MetricCollector is the prometheus.Collector that will be used to store the
	// metric.
	MetricCollector prometheus.Collector
	// ID is a unique identifier for the metric.
	ID string
	// Name is the name of the metric.
	Name string
	// Description is a short description of the metric.
	Description string
	// Type is the type of the metric. It can be one of the following:
	// counter, counter_vec, gauge, gauge_vec, histogram, histogram_vec, summary,
	// summary_vec.
	Type string
	// Args is a list of labels that can be used to distinguish between different
	// dimensions of the same metric.
	Args []string
}

// Prometheus is a middleware that exports Prometheus metrics.
type Prometheus struct {
	reqCnt        *prometheus.CounterVec
	reqDur        *prometheus.HistogramVec
	reqSz, resSz  prometheus.Summary
	router        *gin.Engine
	listenAddress string
	// Ppg is the Prometheus Push Gateway configuration.
	Ppg PrometheusPushGateway

	// MetricsList is a list of custom metrics to be exposed.
	MetricsList []*Metric
	// MetricsPath is the path where the metrics will be exposed.
	MetricsPath string

	// ReqCntURLLabelMappingFn is a function that can be used to map the URL
	// to a different label value.
	ReqCntURLLabelMappingFn RequestCounterURLLabelMappingFn

	// URLLabelFromContext is the name of the context key that will be used
	// to get the URL label from the context.
	URLLabelFromContext string
	CustomLabels        map[string]string
	// DisableBodyReading is a boolean that disables reading the request body.
	DisableBodyReading bool
}

// PrometheusPushGateway contains the configuration for pushing to a Prometheus
// pushgateway.
type PrometheusPushGateway struct {

	// PushIntervalSeconds is the interval at which metrics will be pushed to the
	// pushgateway.
	PushIntervalSeconds time.Duration

	// PushGatewayURL is the URL of the pushgateway.
	PushGatewayURL string

	// MetricsURL is the URL where the metrics are exposed.
	MetricsURL string

	// Job is the job name that will be used when pushing to the pushgateway.
	Job string
}

// Config is a struct for configuring the Prometheus middleware.
type Config struct {
	// Subsystem is the subsystem name to use for the metrics.
	Subsystem string
	// MetricsList is an optional list of custom metrics to be exposed.
	MetricsList []*Metric
	// CustomLabels is a map of custom labels to be added to all metrics.
	CustomLabels map[string]string
	// DisableBodyReading is a boolean that disables reading the request body.
	DisableBodyReading bool
}

// NewPrometheus creates a new Prometheus middleware for backward compatibility.
// It's recommended to use NewWithConfig for new projects.
func NewPrometheus(subsystem string, customMetricsList ...[]*Metric) *Prometheus {
	cfg := Config{
		Subsystem: subsystem,
	}
	if len(customMetricsList) > 0 {
		cfg.MetricsList = customMetricsList[0]
	}
	return NewWithConfig(cfg)
}

// NewWithConfig creates a new Prometheus middleware.
func NewWithConfig(cfg Config) *Prometheus {
	if cfg.Subsystem == "" {
		cfg.Subsystem = "gin"
	}

	copiedStandardMetrics := make([]*Metric, len(standardMetrics))
	for i, m := range standardMetrics {
		newMetric := *m
		newMetric.Args = make([]string, len(m.Args))
		copy(newMetric.Args, m.Args)
		copiedStandardMetrics[i] = &newMetric
	}

	if len(cfg.CustomLabels) > 0 {
		customLabelKeys := make([]string, 0, len(cfg.CustomLabels))
		for k := range cfg.CustomLabels {
			customLabelKeys = append(customLabelKeys, k)
		}

		for _, metric := range copiedStandardMetrics {
			if metric.ID == "reqCnt" || metric.ID == "reqDur" {
				metric.Args = append(metric.Args, customLabelKeys...)
			}
		}
	}

	metricsList := append(cfg.MetricsList, copiedStandardMetrics...)

	p := &Prometheus{
		MetricsList:        metricsList,
		MetricsPath:        defaultMetricPath,
		CustomLabels:       cfg.CustomLabels,
		DisableBodyReading: cfg.DisableBodyReading,
		ReqCntURLLabelMappingFn: func(c *gin.Context) string {
			return c.Request.URL.Path
		},
	}

	p.registerMetrics(cfg.Subsystem)

	return p
}

// SetPushGateway configures the middleware to push metrics to a Prometheus
// pushgateway.
//
// pushGatewayURL is the URL of the pushgateway.
//
// metricsURL is the URL where the metrics are exposed.
//
// pushIntervalSeconds is the interval at which metrics will be pushed to the
// pushgateway.
func (p *Prometheus) SetPushGateway(pushGatewayURL, metricsURL string, pushIntervalSeconds time.Duration) {
	p.Ppg.PushGatewayURL = pushGatewayURL
	p.Ppg.MetricsURL = metricsURL
	p.Ppg.PushIntervalSeconds = pushIntervalSeconds
	p.startPushTicker()
}

// SetPushGatewayJob sets the job name for the pushgateway.
func (p *Prometheus) SetPushGatewayJob(j string) {
	p.Ppg.Job = j
}

// SetListenAddress sets the address where the metrics will be exposed.
func (p *Prometheus) SetListenAddress(address string) {
	p.listenAddress = address
	if p.listenAddress != "" {
		p.router = gin.Default()
		// Log error if router initialization fails
		if p.router == nil {
			log.Error("Failed to initialize gin.Default() router")
		}
	}
}

// SetListenAddressWithRouter sets the address and a custom router where the
// metrics will be exposed.
func (p *Prometheus) SetListenAddressWithRouter(listenAddress string, r *gin.Engine) {
	p.listenAddress = listenAddress
	if len(p.listenAddress) > 0 {
		p.router = r
	}
}

// SetMetricsPath sets the path where the metrics will be exposed.
func (p *Prometheus) SetMetricsPath(e *gin.Engine) {

	if p.listenAddress != "" {
		p.router.GET(p.MetricsPath, prometheusHandler())
		p.runServer()
	} else {
		e.GET(p.MetricsPath, prometheusHandler())
	}
}

// SetMetricsPathWithAuth sets the path where the metrics will be exposed and
// protects it with basic authentication.
func (p *Prometheus) SetMetricsPathWithAuth(e *gin.Engine, accounts gin.Accounts) {

	if p.listenAddress != "" {
		p.router.GET(p.MetricsPath, gin.BasicAuth(accounts), prometheusHandler())
		p.runServer()
	} else {
		e.GET(p.MetricsPath, gin.BasicAuth(accounts), prometheusHandler())
	}

}

func (p *Prometheus) runServer() {
	if p.listenAddress != "" {
		go func() {
			if err := p.router.Run(p.listenAddress); err != nil {
				log.WithError(err).Error("p.router.Run failed")
			}
		}()
	}
}

func (p *Prometheus) getMetrics() []byte {
	response, err := http.Get(p.Ppg.MetricsURL)
	if err != nil {
		log.WithError(err).Error("p.Ppg.MetricsURL failed")
		return []byte{}
	}

	defer func() {
		if err := response.Body.Close(); err != nil {
			log.WithError(err).Error("response.Body.Close failed")
		}
	}()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.WithError(err).Error("io.ReadAll failed")
		return nil
	}

	return body
}

func (p *Prometheus) getPushGatewayURL() string {
	h, err := os.Hostname()
	if err != nil {
		log.WithError(err).Error("os.Hostname failed")
	}
	if p.Ppg.Job == "" {
		p.Ppg.Job = "gin"
	}
	return p.Ppg.PushGatewayURL + "/metrics/job/" + p.Ppg.Job + "/instance/" + h
}

func (p *Prometheus) sendMetricsToPushGateway(metrics []byte) {
	req, err := http.NewRequest("POST", p.getPushGatewayURL(), bytes.NewBuffer(metrics))
	if err != nil {
		log.WithError(err).Errorf("Error creating push gateway request for URL: %s", p.getPushGatewayURL())
		return
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.WithError(err).Errorln("Error sending to push gateway")
		return
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.WithError(err).Error("Error closing push gateway response body")
		}
	}()
}

func (p *Prometheus) startPushTicker() {
	ticker := time.NewTicker(time.Second * p.Ppg.PushIntervalSeconds)
	go func() {
		for range ticker.C {
			p.sendMetricsToPushGateway(p.getMetrics())
		}
	}()
}

// NewMetric creates a new prometheus collector based on the metric type.
func NewMetric(m *Metric, subsystem string) prometheus.Collector {
	var metric prometheus.Collector
	switch m.Type {
	case "counter_vec":
		metric = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Subsystem: subsystem,
				Name:      m.Name,
				Help:      m.Description,
			},
			m.Args,
		)
	case "counter":
		metric = prometheus.NewCounter(
			prometheus.CounterOpts{
				Subsystem: subsystem,
				Name:      m.Name,
				Help:      m.Description,
			},
		)
	case "gauge_vec":
		metric = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Subsystem: subsystem,
				Name:      m.Name,
				Help:      m.Description,
			},
			m.Args,
		)
	case "gauge":
		metric = prometheus.NewGauge(
			prometheus.GaugeOpts{
				Subsystem: subsystem,
				Name:      m.Name,
				Help:      m.Description,
			},
		)
	case "histogram_vec":
		metric = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Subsystem: subsystem,
				Name:      m.Name,
				Help:      m.Description,
			},
			m.Args,
		)
	case "histogram":
		metric = prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Subsystem: subsystem,
				Name:      m.Name,
				Help:      m.Description,
			},
		)
	case "summary_vec":
		metric = prometheus.NewSummaryVec(
			prometheus.SummaryOpts{
				Subsystem: subsystem,
				Name:      m.Name,
				Help:      m.Description,
			},
			m.Args,
		)
	case "summary":
		metric = prometheus.NewSummary(
			prometheus.SummaryOpts{
				Subsystem: subsystem,
				Name:      m.Name,
				Help:      m.Description,
			},
		)
	}
	return metric
}

func (p *Prometheus) registerMetrics(subsystem string) {

	for _, metricDef := range p.MetricsList {
		metric := NewMetric(metricDef, subsystem)
		if err := prometheus.Register(metric); err != nil {
			log.WithError(err).Errorf("%s could not be registered in Prometheus", metricDef.Name)
		}
		switch metricDef.ID {
		case "reqCnt":
			p.reqCnt = metric.(*prometheus.CounterVec)
		case "reqDur":
			p.reqDur = metric.(*prometheus.HistogramVec)
		case "resSz":
			p.resSz = metric.(prometheus.Summary)
		case "reqSz":
			p.reqSz = metric.(prometheus.Summary)
		}
		metricDef.MetricCollector = metric
	}
}

// Use adds the middleware to a gin engine.
func (p *Prometheus) Use(e *gin.Engine) {
	e.Use(p.HandlerFunc())
	p.SetMetricsPath(e)
}

// UseWithAuth adds the middleware to a gin engine with basic authentication.
func (p *Prometheus) UseWithAuth(e *gin.Engine, accounts gin.Accounts) {
	e.Use(p.HandlerFunc())
	p.SetMetricsPathWithAuth(e, accounts)
}

// HandlerFunc returns the gin.HandlerFunc that should be used as a middleware.
func (p *Prometheus) HandlerFunc() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == p.MetricsPath {
			c.Next()
			return
		}

		start := time.Now()
		reqSz := p.computeApproximateRequestSize(c.Request)

		c.Next()

		status := strconv.Itoa(c.Writer.Status())
		elapsed := float64(time.Since(start)) / float64(time.Second)
		resSz := float64(c.Writer.Size())

		url := p.ReqCntURLLabelMappingFn(c)
		// jlambert Oct 2018 - sidecar specific mod
		if len(p.URLLabelFromContext) > 0 {
			u, found := c.Get(p.URLLabelFromContext)
			if !found {
				u = "unknown"
			}
			url = u.(string)
		}
		reqDurLabels := prometheus.Labels{
			"code":   status,
			"method": c.Request.Method,
			"url":    url,
		}
		for k, v := range p.CustomLabels {
			reqDurLabels[k] = v
		}
		p.reqDur.With(reqDurLabels).Observe(elapsed)

		reqCntLabels := prometheus.Labels{
			"code":    status,
			"method":  c.Request.Method,
			"handler": c.HandlerName(),
			"host":    c.Request.Host,
			"url":     url,
		}
		for k, v := range p.CustomLabels {
			reqCntLabels[k] = v
		}
		p.reqCnt.With(reqCntLabels).Inc()

		p.reqSz.Observe(float64(reqSz))
		p.resSz.Observe(resSz)
	}
}

func prometheusHandler() gin.HandlerFunc {
	h := promhttp.Handler()
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

// computeApproximateRequestSize computes the approximate size of a request.
func (p *Prometheus) computeApproximateRequestSize(r *http.Request) int {
	s := 0
	if r.URL != nil {
		s = len(r.URL.Path)
	}

	s += len(r.Method)
	s += len(r.Proto)
	for name, values := range r.Header {
		s += len(name)
		for _, value := range values {
			s += len(value)
		}
	}
	s += len(r.Host)

	// N.B. r.Form and r.MultipartForm are assumed to be included in r.URL.

	if r.Body == nil {
		if r.ContentLength != -1 {
			s += int(r.ContentLength)
		}
		return s
	}

	if p.DisableBodyReading {
		if r.ContentLength != -1 {
			s += int(r.ContentLength)
		}
		return s
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.WithError(err).Error("cannot read request body for size calculation")
		// Fallback to ContentLength if available.
		if r.ContentLength != -1 {
			s += int(r.ContentLength)
		}
		// Try to close the body, but log any error.
		if closeErr := r.Body.Close(); closeErr != nil {
			log.WithError(closeErr).Error("cannot close request body after read error")
		}
		return s
	}

	// Close the original body
	if err := r.Body.Close(); err != nil {
		log.WithError(err).Error("cannot close request body after reading for size calculation")
	}

	// Add body size to total
	s += len(bodyBytes)

	// Replace the body so it can be read again by subsequent handlers
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	return s
}
