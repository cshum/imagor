package instrumentation

import (
	"context"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

var (
	// MethodLatency tracks latency for individual methods
	MethodLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "imagor_method_duration_seconds",
			Help:    "A histogram of latencies for individual methods",
			Buckets: []float64{.001, .0025, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		},
		[]string{"package", "struct", "method", "status"},
	)

	// MethodCounter tracks method call counts
	MethodCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "imagor_method_calls_total",
			Help: "Total number of method calls",
		},
		[]string{"package", "struct", "method", "status"},
	)
)

func init() {
	prometheus.MustRegister(MethodLatency)
	prometheus.MustRegister(MethodCounter)
}

// Instrumentation provides method-level metrics tracking
type Instrumentation struct {
	Logger *zap.Logger
}

// New creates a new Instrumentation instance
func New(logger *zap.Logger) *Instrumentation {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Instrumentation{
		Logger: logger,
	}
}

// MethodTimer provides a convenient timer for method instrumentation
type MethodTimer struct {
	instrumentation *Instrumentation
	pkg             string
	structName      string
	methodName      string
	start           time.Time
}

// NewMethodTimer creates a new method timer
func (i *Instrumentation) NewMethodTimer(pkg, structName, methodName string) *MethodTimer {
	return &MethodTimer{
		instrumentation: i,
		pkg:             pkg,
		structName:      structName,
		methodName:      methodName,
		start:           time.Now(),
	}
}

// NewMethodTimerFromString creates a new method timer from a single string identifier
// Format: "package.struct.method" or "struct.method" (defaults to "imagor" package)
func (i *Instrumentation) NewMethodTimerFromString(identifier string) *MethodTimer {
	parts := strings.Split(identifier, ".")
	var pkg, structName, methodName string

	switch len(parts) {
	case 2:
		// "struct.method" format
		pkg = "imagor"
		structName = parts[0]
		methodName = parts[1]
	case 3:
		// "package.struct.method" format
		pkg = parts[0]
		structName = parts[1]
		methodName = parts[2]
	default:
		// Fallback to treating the whole string as method name
		pkg = "imagor"
		structName = "Unknown"
		methodName = identifier
	}

	return &MethodTimer{
		instrumentation: i,
		pkg:             pkg,
		structName:      structName,
		methodName:      methodName,
		start:           time.Now(),
	}
}

// ObserveDuration records the duration and metrics for the method
func (mt *MethodTimer) ObserveDuration() {
	if mt.instrumentation == nil {
		return
	}
	duration := time.Since(mt.start)
	mt.instrumentation.RecordMethodDuration(context.Background(), mt.pkg, mt.structName, mt.methodName, duration, nil)
}

// ObserveDurationWithError records the duration and metrics for the method with an error
func (mt *MethodTimer) ObserveDurationWithError(err error) {
	if mt.instrumentation == nil {
		return
	}
	duration := time.Since(mt.start)
	mt.instrumentation.RecordMethodDuration(context.Background(), mt.pkg, mt.structName, mt.methodName, duration, err)
}

// RecordMethodDuration records the duration and status of a method call
func (i *Instrumentation) RecordMethodDuration(ctx context.Context, pkg, structName, methodName string, duration time.Duration, err error) {
	status := "success"
	if err != nil {
		status = "error"
	}

	// Record metrics
	MethodLatency.WithLabelValues(pkg, structName, methodName, status).Observe(duration.Seconds())
	MethodCounter.WithLabelValues(pkg, structName, methodName, status).Inc()

	// Log if debug is enabled
	if i.Logger != nil {
		i.Logger.Debug("method_execution",
			zap.String("package", pkg),
			zap.String("struct", structName),
			zap.String("method", methodName),
			zap.Duration("duration", duration),
			zap.String("status", status),
			zap.Error(err))
	}
}
