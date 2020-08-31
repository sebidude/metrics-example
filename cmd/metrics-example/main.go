package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	hostname       string
	greeting       string
	requestCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "metrics_example_request_counter",
		Help: "Counter of the requests for the hello endpoint",
	}, []string{"code", "method", "endpoint"})
	requestHistogram = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "metrics_example_request_latency",
		Help:    "Latency of the requests for the hello endpoint",
		Buckets: []float64{0.005, 0.01, 0.05, 0.1, 0.3, 0.5, 0.9, 1, 2, 5},
	}, []string{"code", "method", "endpoint"})
)

func GetEnvOrDefault(key, defaultValue string) string {
	v := os.Getenv(key)
	if len(v) > 0 {
		return v
	}
	return defaultValue
}

func main() {
	var err error
	hostname, err = os.Hostname()
	if err != nil {
		fmt.Println("Cannot get hostname!")
		os.Exit(1)
	}

	greeting = GetEnvOrDefault("GREETING", "simple webtest")
	prometheus.MustRegister(requestCounter)
	prometheus.MustRegister(requestHistogram)

	r := gin.New()
	r.Use(gin.Recovery())

	metrics := r.Group("/metrics")
	metrics.GET("", prometheusHandler())

	hello := r.Group("/hello")
	hello.Use(gin.Logger())
	hello.Use(MetricsMiddleware())
	hello.Any("", helloHandler)
	hello.Any("*path", helloHandler)

	probe := r.Group("/probe")
	probe.GET("/ready", readyProbe)
	probe.GET("/alive", aliveProbe)

	r.Run(GetEnvOrDefault("LISTEN_ADDRESS", ":8080"))
}

func readyProbe(c *gin.Context) {
	c.String(200, "Ready.")
}

func aliveProbe(c *gin.Context) {
	c.String(200, "Alive.")
}

func helloHandler(c *gin.Context) {
	path := c.Param("path")
	if path == "/fail" {
		c.String(500, "Fail Hello World from %s (%s): %s", hostname, greeting, path)
		return
	}
	c.String(200, "Hello World from %s (%s): %s", hostname, greeting, path)
}

func prometheusHandler() gin.HandlerFunc {
	h := promhttp.Handler()
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		t := time.Now()
		c.Next()
		latency := time.Since(t)

		// access the status we are sending
		status := c.Writer.Status()
		requestCounter.WithLabelValues(strconv.Itoa(status), c.Request.Method, c.Request.RequestURI).Inc()
		requestHistogram.WithLabelValues(strconv.Itoa(status), c.Request.Method, c.Request.RequestURI).Observe(latency.Seconds())
	}
}
