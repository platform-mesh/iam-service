package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/pkg/errors"
	"github.com/platform-mesh/golang-commons/context"
	"github.com/platform-mesh/golang-commons/logger"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const publicErrorMessage = "Error while validating token"

// TokenReview structs based on Kubernetes API
type TokenReview struct {
	APIVersion string            `json:"apiVersion"`
	Kind       string            `json:"kind"`
	Spec       TokenReviewSpec   `json:"spec"`
	Status     TokenReviewStatus `json:"status,omitempty"`
}

type TokenReviewSpec struct {
	Token     string   `json:"token"`
	Audiences []string `json:"audiences,omitempty"`
}

type TokenReviewStatus struct {
	Authenticated bool   `json:"authenticated"`
	Error         string `json:"error,omitempty"`
	User          *User  `json:"user,omitempty"`
}

type User struct {
	Username string              `json:"username,omitempty"`
	UID      string              `json:"uid,omitempty"`
	Groups   []string            `json:"groups,omitempty"`
	Extra    map[string][]string `json:"extra,omitempty"`
}

type KCPValidation struct {
	restConfig *rest.Config
}

func NewKCPValidation(kcpKubeconfig string) (*KCPValidation, error) {
	apiCfg, err := clientcmd.LoadFromFile(kcpKubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig from %s: %w", kcpKubeconfig, err)
	}
	restCfg, err := clientcmd.NewDefaultClientConfig(*apiCfg, &clientcmd.ConfigOverrides{}).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create rest config: %w", err)
	}

	return &KCPValidation{restConfig: restCfg}, nil
}

func (k *KCPValidation) ValidateTokenHandler() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			log := logger.LoadLoggerFromContext(ctx)

			// get tenantId from context
			tenantId, err := context.GetTenantFromContext(ctx)
			if err != nil {
				log.Error().Err(errors.WithStack(err)).Msg("Error while retrieving the tenant from the context")
				http.Error(w, "Error while retrieving the tenant from the context", http.StatusInternalServerError)
				return
			}
			log = log.ChildLogger("tenantId", tenantId)
			tenantParts := strings.Split(tenantId, "/")
			clusterId := tenantParts[0]

			// get token from context
			token, err := context.GetAuthHeaderFromContext(ctx)
			if err != nil {
				log.Error().Err(errors.WithStack(err)).Msg("Error while retrieving the token from the context")
				http.Error(w, "Error while retrieving the token from the context", http.StatusInternalServerError)
				return
			}

			// call to kcp by creating a TokenReview Request against the KCP URL
			log.Debug().Msg("Validating token using TokenReview API")

			// Create TokenReview request
			tokenReview := TokenReview{
				APIVersion: "authentication.k8s.io/v1",
				Kind:       "TokenReview",
				Spec: TokenReviewSpec{
					Token: token,
				},
			}

			// Marshal the TokenReview request to JSON
			requestBody, err := json.Marshal(tokenReview)
			if err != nil {
				log.Error().Err(errors.WithStack(err)).Msg("Error marshaling TokenReview request")
				http.Error(w, publicErrorMessage, http.StatusInternalServerError)
				return
			}

			// Use TokenReview API endpoint
			clusterUrl, err := url.Parse(k.restConfig.Host)
			if err != nil {
				log.Error().Err(errors.WithStack(err)).Msg("Error parsing KCP host URL")
			}

			apiURL := fmt.Sprintf("%s://%s/clusters/%s/apis/authentication.k8s.io/v1/tokenreviews", clusterUrl.Scheme, clusterUrl.Host, clusterId)
			req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(requestBody))
			if err != nil {
				log.Error().Err(errors.WithStack(err)).Msg("Error creating TokenReview HTTP request")
				http.Error(w, publicErrorMessage, http.StatusInternalServerError)
				return
			}

			// Set Content-Type header for JSON
			req.Header.Set("Content-Type", "application/json")

			httpClient, err := rest.HTTPClientFor(k.restConfig)
			if err != nil {
				log.Error().Err(errors.WithStack(err)).Msg("Error creating httpClient")
				http.Error(w, publicErrorMessage, http.StatusInternalServerError)
				return
			}

			resp, err := httpClient.Do(req)
			if err != nil {
				log.Error().Err(errors.WithStack(err)).Msg("TokenReview request failed")
				http.Error(w, publicErrorMessage, http.StatusInternalServerError)
				return
			}
			defer func(Body io.ReadCloser) {
				err := Body.Close()
				if err != nil {
					log.Error().Err(errors.WithStack(err)).Msg("Error closing response body")
				}
			}(resp.Body)

			log.Debug().Int("status", resp.StatusCode).Msg("TokenReview response received")

			// Check HTTP response status
			if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
				log.Debug().Int("status", resp.StatusCode).Msg("TokenReview request failed with unexpected status")
				http.Error(w, publicErrorMessage, http.StatusInternalServerError)
				return
			}

			// Parse the TokenReview response
			responseBody, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Error().Err(errors.WithStack(err)).Msg("Error reading TokenReview response body")
				http.Error(w, publicErrorMessage, http.StatusInternalServerError)
				return
			}

			var tokenReviewResponse TokenReview
			if err := json.Unmarshal(responseBody, &tokenReviewResponse); err != nil {
				log.Error().Err(errors.WithStack(err)).Msg("Error unmarshaling TokenReview response")
				http.Error(w, publicErrorMessage, http.StatusInternalServerError)
				return
			}

			// Check if token is authenticated
			if !tokenReviewResponse.Status.Authenticated {
				if tokenReviewResponse.Status.Error != "" {
					log.Debug().Str("error", tokenReviewResponse.Status.Error).Msg("Token validation failed with error")
				} else {
					log.Debug().Msg("Token validation failed - not authenticated")
				}
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}

			// Token is valid and authenticated
			log.Debug().Msg("Token validation successful")
			if tokenReviewResponse.Status.User != nil {
				log.Debug().Str("username", tokenReviewResponse.Status.User.Username).Msg("Authenticated user")
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
