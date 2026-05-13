package metrics_test

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/suite"

	"github.com/platform-mesh/iam-service/pkg/metrics"
)

type MetricsTestSuite struct {
	suite.Suite
}

func TestMetricsTestSuite(t *testing.T) {
	suite.Run(t, new(MetricsTestSuite))
}

// TestAuthorizationChecks verifies that the AuthorizationChecks counter increments
// correctly for each result label (allowed/denied/error).
func (s *MetricsTestSuite) TestAuthorizationChecks() {
	before := testutil.ToFloat64(metrics.AuthorizationChecks.WithLabelValues("allowed"))
	metrics.AuthorizationChecks.WithLabelValues("allowed").Inc()
	s.Require().Equal(before+1, testutil.ToFloat64(metrics.AuthorizationChecks.WithLabelValues("allowed")))

	before = testutil.ToFloat64(metrics.AuthorizationChecks.WithLabelValues("denied"))
	metrics.AuthorizationChecks.WithLabelValues("denied").Inc()
	s.Require().Equal(before+1, testutil.ToFloat64(metrics.AuthorizationChecks.WithLabelValues("denied")))

	before = testutil.ToFloat64(metrics.AuthorizationChecks.WithLabelValues("error"))
	metrics.AuthorizationChecks.WithLabelValues("error").Inc()
	s.Require().Equal(before+1, testutil.ToFloat64(metrics.AuthorizationChecks.WithLabelValues("error")))
}

// TestAuthorizationDuration verifies that the AuthorizationDuration histogram records
// observations per permission label.
func (s *MetricsTestSuite) TestAuthorizationDuration() {
	before := testutil.CollectAndCount(metrics.AuthorizationDuration)
	metrics.AuthorizationDuration.WithLabelValues("read").Observe(0.05)
	s.Assert().Greater(testutil.CollectAndCount(metrics.AuthorizationDuration), before)
}

// TestGraphQLRequests verifies that the GraphQLRequests counter increments
// correctly for each operation/result label combination.
func (s *MetricsTestSuite) TestGraphQLRequests() {
	before := testutil.ToFloat64(metrics.GraphQLRequests.WithLabelValues("Users", "success"))
	metrics.GraphQLRequests.WithLabelValues("Users", "success").Inc()
	s.Require().Equal(before+1, testutil.ToFloat64(metrics.GraphQLRequests.WithLabelValues("Users", "success")))

	before = testutil.ToFloat64(metrics.GraphQLRequests.WithLabelValues("AssignRolesToUsers", "error"))
	metrics.GraphQLRequests.WithLabelValues("AssignRolesToUsers", "error").Inc()
	s.Require().Equal(before+1, testutil.ToFloat64(metrics.GraphQLRequests.WithLabelValues("AssignRolesToUsers", "error")))
}

// TestKeycloakRequests verifies that the KeycloakRequests counter increments
// correctly for each operation/result label combination.
func (s *MetricsTestSuite) TestKeycloakRequests() {
	before := testutil.ToFloat64(metrics.KeycloakRequests.WithLabelValues("user_by_mail", "success"))
	metrics.KeycloakRequests.WithLabelValues("user_by_mail", "success").Inc()
	s.Require().Equal(before+1, testutil.ToFloat64(metrics.KeycloakRequests.WithLabelValues("user_by_mail", "success")))

	before = testutil.ToFloat64(metrics.KeycloakRequests.WithLabelValues("get_users", "error"))
	metrics.KeycloakRequests.WithLabelValues("get_users", "error").Inc()
	s.Require().Equal(before+1, testutil.ToFloat64(metrics.KeycloakRequests.WithLabelValues("get_users", "error")))
}

// TestKeycloakDuration verifies that the KeycloakDuration histogram records
// observations per operation label.
func (s *MetricsTestSuite) TestKeycloakDuration() {
	before := testutil.CollectAndCount(metrics.KeycloakDuration)
	metrics.KeycloakDuration.WithLabelValues("get_users").Observe(0.12)
	s.Assert().Greater(testutil.CollectAndCount(metrics.KeycloakDuration), before)
}
