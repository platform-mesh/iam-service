package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	ctrlmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	// AuthorizationChecks counts OpenFGA permission checks by result (allowed/denied/error).
	AuthorizationChecks = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "iam_authorization_checks_total",
			Help: "Total number of OpenFGA authorization checks by result.",
		},
		[]string{"result"},
	)

	// AuthorizationDuration observes how long each OpenFGA check takes, labelled by permission.
	AuthorizationDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "iam_authorization_duration_seconds",
			Help:    "Duration of OpenFGA authorization checks in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"permission"},
	)

	// GraphQLRequests counts incoming GraphQL operations by operation name and result.
	GraphQLRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "iam_graphql_requests_total",
			Help: "Total number of GraphQL operations by operation name and result.",
		},
		[]string{"operation", "result"},
	)

	// KeycloakRequests counts outgoing Keycloak API calls by operation and result.
	KeycloakRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "iam_keycloak_requests_total",
			Help: "Total number of Keycloak API calls by operation and result.",
		},
		[]string{"operation", "result"},
	)

	// KeycloakDuration observes how long each Keycloak API call takes, labelled by operation.
	KeycloakDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "iam_keycloak_duration_seconds",
			Help:    "Duration of Keycloak API calls in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation"},
	)
)

func init() {
	ctrlmetrics.Registry.MustRegister(
		AuthorizationChecks,
		AuthorizationDuration,
		GraphQLRequests,
		KeycloakRequests,
		KeycloakDuration,
	)
}
