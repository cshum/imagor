package storage

import "github.com/prometheus/client_golang/prometheus"

var OperationHistorgram = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name: "storage_operations_total",
		Help: "A histogram of storage operations",
	},
	[]string{"storage", "operation"},
)

func init() {
	prometheus.MustRegister(OperationHistorgram)
}
