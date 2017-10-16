package ginprometheus

import (
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestNewPrometheus(t *testing.T) {
	newProm := NewPrometheus("passionfruit")
	expectedProm := &Prometheus{MetricsPath: defaultMetricPath}
	assert.Equal(t, expectedProm.MetricsPath, newProm.MetricsPath)
}

func TestSetPushGatewayJob(t *testing.T) {
	p := NewPrometheus("icemelts")
	p.SetPushGatewayJob("twobirdsonestone")
	assert.Equal(t, "twobirdsonestone", p.Ppg.Job)
}

func TestRegisterMetrics(t *testing.T) {
	p := NewPrometheus("portland")
	pCopy := NewPrometheus("portland")
	assert.NotEqual(t, p, pCopy)
}

func TestSetListenAddress(t *testing.T) {
	p := NewPrometheus("kmt")
	p.SetListenAddress("localhost:4422")
	assert.Equal(t, p.listenAddress, "localhost:4422")
}

func TestSetPushGateway(t *testing.T) {
	p := NewPrometheus("glow")
	p.SetPushGateway("pushGatewayURL.com", "metricsURL.com", 5*time.Minute)
	assert.Equal(t, "pushGatewayURL.com", p.Ppg.PushGatewayURL)
	assert.Equal(t, "metricsURL.com", p.Ppg.MetricsURL)
	assert.Equal(t, 5*time.Minute, p.Ppg.PushIntervalSeconds)
}

func TestGetPushGatewayURL(t *testing.T) {
	p := NewPrometheus("glow")
	p.SetPushGateway("pushGatewayURL.com", "metricsURL.com", 5*time.Minute)
	host, _ := os.Hostname()
	pushgateway := p.Ppg.PushGatewayURL + "/metrics/job/gin/instance/" + host
	assert.Equal(t, pushgateway, p.getPushGatewayURL())
}

func TestUse(t *testing.T) {
	r := gin.New()
	p := NewPrometheus("gin")
	p.Use(r)
	assert.Equal(t, r.Routes()[0].Path, "/metrics")
}
