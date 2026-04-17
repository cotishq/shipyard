package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	QueueDepth = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "shipyard",
		Name:      "queue_depth",
		Help:      "Number of deployments currently in QUEUED state.",
	})

	ActiveBuilds = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "shipyard",
		Name:      "active_builds",
		Help:      "Number of deployments currently in BUILDING state.",
	})

	BuildDurationSeconds = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "shipyard",
		Name:      "build_duration_seconds",
		Help:      "Duration of successful deployment builds in seconds.",
		Buckets:   prometheus.DefBuckets,
	})

	DeploymentSuccessTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "shipyard",
		Name:      "deployment_success_total",
		Help:      "Total number of successful deployments.",
	})

	DeploymentFailureTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "shipyard",
		Name:      "deployment_failure_total",
		Help:      "Total number of failed deployments.",
	})

	RetryTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "shipyard",
		Name:      "retry_total",
		Help:      "Total number of deployment retries.",
	})

	ArtifactUploadFailureTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "shipyard",
		Name:      "artifact_upload_failure_total",
		Help:      "Total number of deployment artifact upload failures.",
	})

	DeployRequestsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: "shipyard",
		Name:      "deploy_requests_total",
		Help:      "Total number of deployment trigger requests.",
	})

	WebhookRequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "shipyard",
		Name:      "webhook_requests_total",
		Help:      "Total number of webhook requests by provider and result.",
	}, []string{"provider", "result"})
)

func Init() {
	prometheus.MustRegister(
		QueueDepth,
		ActiveBuilds,
		BuildDurationSeconds,
		DeploymentSuccessTotal,
		DeploymentFailureTotal,
		RetryTotal,
		ArtifactUploadFailureTotal,
		DeployRequestsTotal,
		WebhookRequestsTotal,
	)
}

func SetQueueDepth(n int) {
	QueueDepth.Set(float64(n))
}

func SetActiveBuilds(n int) {
	ActiveBuilds.Set(float64(n))
}

func ObserveBuildDuration(seconds float64) {
	BuildDurationSeconds.Observe(seconds)
}

func IncDeploymentSuccess() {
	DeploymentSuccessTotal.Inc()
}

func IncDeploymentFailure() {
	DeploymentFailureTotal.Inc()
}

func IncRetry() {
	RetryTotal.Inc()
}

func IncArtifactUploadFailure() {
	ArtifactUploadFailureTotal.Inc()
}

func IncDeployRequest() {
	DeployRequestsTotal.Inc()
}

func IncWebhookRequest(provider, result string) {
	WebhookRequestsTotal.WithLabelValues(provider, result).Inc()
}
