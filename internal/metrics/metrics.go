package metrics

import (
	"runtime"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HTTP metrics
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ovncp_http_requests_total",
			Help: "Total number of HTTP requests processed",
		},
		[]string{"method", "endpoint", "status"},
	)

	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ovncp_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	HTTPRequestSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ovncp_http_request_size_bytes",
			Help:    "HTTP request size in bytes",
			Buckets: prometheus.ExponentialBuckets(100, 10, 6),
		},
		[]string{"method", "endpoint"},
	)

	HTTPResponseSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ovncp_http_response_size_bytes",
			Help:    "HTTP response size in bytes",
			Buckets: prometheus.ExponentialBuckets(100, 10, 6),
		},
		[]string{"method", "endpoint"},
	)

	// OVN operation metrics
	OVNOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ovncp_ovn_operations_total",
			Help: "Total number of OVN operations",
		},
		[]string{"operation", "resource", "status"},
	)

	OVNOperationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ovncp_ovn_operation_duration_seconds",
			Help:    "OVN operation duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation", "resource"},
	)

	OVNConnectionStatus = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ovncp_ovn_connection_status",
			Help: "OVN connection status (1=connected, 0=disconnected)",
		},
		[]string{"component"}, // northbound, southbound
	)

	// Database metrics
	DBQueriesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ovncp_db_queries_total",
			Help: "Total number of database queries",
		},
		[]string{"query_type", "table", "status"},
	)

	DBQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "ovncp_db_query_duration_seconds",
			Help:    "Database query duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"query_type", "table"},
	)

	DBConnectionsActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "ovncp_db_connections_active",
			Help: "Number of active database connections",
		},
	)

	DBConnectionsIdle = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "ovncp_db_connections_idle",
			Help: "Number of idle database connections",
		},
	)

	// Business metrics
	LogicalSwitchesTotal = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "ovncp_logical_switches_total",
			Help: "Total number of logical switches",
		},
	)

	LogicalRoutersTotal = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "ovncp_logical_routers_total",
			Help: "Total number of logical routers",
		},
	)

	LogicalPortsTotal = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "ovncp_logical_ports_total",
			Help: "Total number of logical ports",
		},
	)

	ACLRulesTotal = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "ovncp_acl_rules_total",
			Help: "Total number of ACL rules",
		},
	)

	LoadBalancersTotal = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "ovncp_load_balancers_total",
			Help: "Total number of load balancers",
		},
	)

	// Authentication metrics
	AuthRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ovncp_auth_requests_total",
			Help: "Total number of authentication requests",
		},
		[]string{"method", "status"}, // method: jwt, api_key, oauth
	)

	ActiveSessionsTotal = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "ovncp_active_sessions_total",
			Help: "Total number of active user sessions",
		},
	)

	// Transaction metrics
	TransactionsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ovncp_transactions_total",
			Help: "Total number of OVN transactions",
		},
		[]string{"status"}, // success, failure, rollback
	)

	TransactionOperationsHistogram = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "ovncp_transaction_operations_count",
			Help:    "Number of operations per transaction",
			Buckets: prometheus.LinearBuckets(1, 5, 10),
		},
	)

	// Cache metrics
	CacheHitsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ovncp_cache_hits_total",
			Help: "Total number of cache hits",
		},
		[]string{"cache_name"},
	)

	CacheMissesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ovncp_cache_misses_total",
			Help: "Total number of cache misses",
		},
		[]string{"cache_name"},
	)

	CacheEvictionsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ovncp_cache_evictions_total",
			Help: "Total number of cache evictions",
		},
		[]string{"cache_name"},
	)

	// Error metrics
	ErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ovncp_errors_total",
			Help: "Total number of errors",
		},
		[]string{"component", "error_type"},
	)

	PanicsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "ovncp_panics_total",
			Help: "Total number of panics recovered",
		},
	)

	// System metrics
	GoroutinesCount = promauto.NewGaugeFunc(
		prometheus.GaugeOpts{
			Name: "ovncp_goroutines_count",
			Help: "Current number of goroutines",
		},
		func() float64 {
			return float64(runtime.NumGoroutine())
		},
	)

	MemoryUsageBytes = promauto.NewGaugeFunc(
		prometheus.GaugeOpts{
			Name: "ovncp_memory_usage_bytes",
			Help: "Current memory usage in bytes",
		},
		func() float64 {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			return float64(m.Alloc)
		},
	)

	// Build info
	BuildInfo = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ovncp_build_info",
			Help: "Build information",
		},
		[]string{"version", "commit", "build_time"},
	)
)

// Helper functions for recording metrics

// RecordHTTPRequest records HTTP request metrics
func RecordHTTPRequest(method, endpoint, status string, duration float64, requestSize, responseSize int64) {
	HTTPRequestsTotal.WithLabelValues(method, endpoint, status).Inc()
	HTTPRequestDuration.WithLabelValues(method, endpoint).Observe(duration)
	if requestSize > 0 {
		HTTPRequestSize.WithLabelValues(method, endpoint).Observe(float64(requestSize))
	}
	if responseSize > 0 {
		HTTPResponseSize.WithLabelValues(method, endpoint).Observe(float64(responseSize))
	}
}

// RecordOVNOperation records OVN operation metrics
func RecordOVNOperation(operation, resource, status string, duration float64) {
	OVNOperationsTotal.WithLabelValues(operation, resource, status).Inc()
	OVNOperationDuration.WithLabelValues(operation, resource).Observe(duration)
}

// RecordDBQuery records database query metrics
func RecordDBQuery(queryType, table, status string, duration float64) {
	DBQueriesTotal.WithLabelValues(queryType, table, status).Inc()
	DBQueryDuration.WithLabelValues(queryType, table).Observe(duration)
}

// RecordAuth records authentication metrics
func RecordAuth(method, status string) {
	AuthRequestsTotal.WithLabelValues(method, status).Inc()
}

// RecordTransaction records transaction metrics
func RecordTransaction(status string, operationCount int) {
	TransactionsTotal.WithLabelValues(status).Inc()
	TransactionOperationsHistogram.Observe(float64(operationCount))
}

// RecordCacheOperation records cache operation metrics
func RecordCacheOperation(cacheName string, hit bool) {
	if hit {
		CacheHitsTotal.WithLabelValues(cacheName).Inc()
	} else {
		CacheMissesTotal.WithLabelValues(cacheName).Inc()
	}
}

// RecordError records error metrics
func RecordError(component, errorType string) {
	ErrorsTotal.WithLabelValues(component, errorType).Inc()
}

// UpdateResourceCounts updates resource count gauges
func UpdateResourceCounts(switches, routers, ports, acls, loadBalancers float64) {
	LogicalSwitchesTotal.Set(switches)
	LogicalRoutersTotal.Set(routers)
	LogicalPortsTotal.Set(ports)
	ACLRulesTotal.Set(acls)
	LoadBalancersTotal.Set(loadBalancers)
}

// SetOVNConnectionStatus sets OVN connection status
func SetOVNConnectionStatus(component string, connected bool) {
	value := float64(0)
	if connected {
		value = 1
	}
	OVNConnectionStatus.WithLabelValues(component).Set(value)
}

// UpdateDBConnectionMetrics updates database connection pool metrics
func UpdateDBConnectionMetrics(active, idle int) {
	DBConnectionsActive.Set(float64(active))
	DBConnectionsIdle.Set(float64(idle))
}