package metrics

import (
	"expo-open-ota/internal/cache"
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// All metrics are scoped by appId. In multi-app deployments (v2), two
// different apps can publish identically named branches / runtime versions,
// so we include appId in the label set AND in the Redis cache keys — if we
// didn't, the seen-users sets would merge across apps and skew the unique
// counts.
var (
	activeUsersVec = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "active_users_total",
			Help: "Total number of unique active users per appId, platform, runtime version, branch and update",
		},
		[]string{"appId", "platform", "runtime", "branch", "update"},
	)

	globalActiveUsersVec = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "global_active_users_total",
			Help: "Total number of unique active users per appId and platform across all runtime versions, branches and updates",
		},
		[]string{"appId", "platform"},
	)

	updateDownloadsVec = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "update_downloads_total",
			Help: "Total number of update downloads per appId, platform, runtime version, branch and update",
		},
		[]string{"appId", "platform", "runtime", "branch", "update", "updateType"},
	)

	updateErrorUsersVec = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "update_error_users_total",
			Help: "Total number of users who encountered an error for a given appId, platform, runtime version, branch and update",
		},
		[]string{"appId", "platform", "runtime", "branch", "update"},
	)
)

func InitMetrics() {
	prometheus.MustRegister(activeUsersVec)
	prometheus.MustRegister(updateDownloadsVec)
	prometheus.MustRegister(updateErrorUsersVec)
	prometheus.MustRegister(globalActiveUsersVec)
}

func CleanupMetrics() {
	prometheus.Unregister(activeUsersVec)
	prometheus.Unregister(updateDownloadsVec)
	prometheus.Unregister(updateErrorUsersVec)
	prometheus.Unregister(globalActiveUsersVec)
}

func TrackUpdateErrorUsers(appId, clientId, platform, runtime, branch, update string) {
	computedUpdate := update
	if computedUpdate == "" {
		computedUpdate = "unknown"
	}
	if appId == "" || clientId == "" || platform == "" || runtime == "" || branch == "" {
		return
	}
	resolvedCache := cache.GetCache()
	key := fmt.Sprintf("update_error_users:%s:%s:%s:%s:%s", appId, branch, platform, runtime, computedUpdate)
	ttl := 600

	_ = resolvedCache.Sadd(key, []string{clientId}, &ttl)

	count, err := resolvedCache.Scard(key)
	if err != nil {
		return
	}
	updateErrorUsersVec.WithLabelValues(appId, platform, runtime, branch, computedUpdate).Set(float64(count))
}

func TrackActiveUser(appId, clientId, platform, runtime, branch, update string) {
	if appId == "" || clientId == "" || platform == "" || branch == "" || update == "" || runtime == "" {
		return
	}

	resolvedCache := cache.GetCache()
	activeUserKey := fmt.Sprintf("seen_users:%s:%s:%s:%s:%s", appId, branch, platform, runtime, update)
	ttl := 14400

	_ = resolvedCache.Sadd(activeUserKey, []string{clientId}, &ttl)

	count, err := resolvedCache.Scard(activeUserKey)
	if err != nil {
		return
	}
	activeUsersVec.WithLabelValues(appId, platform, runtime, branch, update).Set(float64(count))

	globalActiveUserKey := fmt.Sprintf("global_active_users:%s:%s", appId, platform)
	_ = resolvedCache.Sadd(globalActiveUserKey, []string{clientId}, &ttl)
	count, err = resolvedCache.Scard(globalActiveUserKey)
	if err != nil {
		return
	}
	globalActiveUsersVec.WithLabelValues(appId, platform).Set(float64(count))
}

func TrackUpdateDownload(appId, platform, runtime, branch, update, updateType string) {
	if appId == "" || update == "" || platform == "" || branch == "" {
		return
	}
	updateDownloadsVec.WithLabelValues(appId, platform, runtime, branch, update, updateType).Inc()
}

func PrometheusHandler() http.Handler {
	return promhttp.Handler()
}

func ResetMetricsForTest() {
	activeUsersVec = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "active_users_total",
			Help: "Total number of unique active users per appId, platform, runtime version, branch and update",
		},
		[]string{"appId", "platform", "runtime", "branch", "update"},
	)
	updateDownloadsVec = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "update_downloads_total",
			Help: "Total number of update downloads per appId, platform, runtime version, branch and update",
		},
		[]string{"appId", "platform", "runtime", "branch", "update", "updateType"},
	)
	updateErrorUsersVec = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "update_error_users_total",
			Help: "Total number of users who encountered an error for a given appId, platform, runtime version, branch and update",
		},
		[]string{"appId", "platform", "runtime", "branch", "update"},
	)
	globalActiveUsersVec = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "global_active_users_total",
			Help: "Total number of unique active users per appId and platform across all runtime versions, branches and updates",
		},
		[]string{"appId", "platform"},
	)
}
