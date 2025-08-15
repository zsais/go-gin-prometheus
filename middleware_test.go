package ginprometheus

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
)

func TestPrometheusMiddleware(t *testing.T) {
	reg := prometheus.NewRegistry()
	prometheus.DefaultRegisterer = reg
	prometheus.DefaultGatherer = reg
	r := gin.New()
	p := NewWithConfig(Config{})
	p.Use(r)

	r.GET("/api/v1/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	// Test that the middleware sets the request counter
	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Check that the metrics endpoint is working
	req = httptest.NewRequest("GET", "/metrics", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d but got %d", http.StatusOK, w.Code)
	}
	if !strings.Contains(w.Body.String(), "requests_total") {
		t.Errorf("expected requests_total metric but it was not found")
	}
}

func TestCustomLabels(t *testing.T) {
	reg := prometheus.NewRegistry()
	prometheus.DefaultRegisterer = reg
	prometheus.DefaultGatherer = reg
	r := gin.New()
	p := NewWithConfig(Config{
		CustomLabels: map[string]string{"custom_label": "test_value"},
	})
	p.Use(r)

	r.GET("/api/v1/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest("GET", "/api/v1/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Check that the metrics endpoint is working and contains the custom label
	req = httptest.NewRequest("GET", "/metrics", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d but got %d", http.StatusOK, w.Code)
	}

	if !strings.Contains(w.Body.String(), "custom_label=\"test_value\"") {
		t.Errorf("expected custom label to be set but it was not")
	}
}

func TestDisableBodyReading(t *testing.T) {
	reg := prometheus.NewRegistry()
	prometheus.DefaultRegisterer = reg
	prometheus.DefaultGatherer = reg
	r := gin.New()
	p := NewWithConfig(Config{
		DisableBodyReading: true,
	})
	p.Use(r)

	r.POST("/api/v1/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	// Test that the middleware sets the request counter
	req := httptest.NewRequest("POST", "/api/v1/test", strings.NewReader("test"))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Check that the metrics endpoint is working
	req = httptest.NewRequest("GET", "/metrics", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d but got %d", http.StatusOK, w.Code)
	}
	if strings.Contains(w.Body.String(), "request_size_bytes_sum{quantile=\"0.5\"} 4") {
		t.Errorf("expected request_size_bytes_sum to not include body size")
	}
}