package middleware

import (
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
	"github.com/platform-mesh/golang-commons/context"
	"github.com/platform-mesh/golang-commons/logger"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const publicErrorMessage = "Error while validating token"

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

func (k *KCPValidation) validateTokenHandler() func(http.Handler) http.Handler {
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

			// get token from context
			token, err := context.GetAuthHeaderFromContext(ctx)
			if err != nil {
				log.Error().Err(errors.WithStack(err)).Msg("Error while retrieving the token from the context")
				http.Error(w, "Error while retrieving the token from the context", http.StatusInternalServerError)
				return
			}

			// call to kcp by creating a TokenReview Request against the KCP URL
			log.Debug().Msg("Validating token for introspection query")

			// Use namespaces endpoint for token validation - it's a resource endpoint (not discovery)
			// so it will use the token authentication instead of being routed to admin credentials
			apiURL, err := url.JoinPath(k.restConfig.Host, "/api/v1/namespaces")
			if err != nil {
				log.Error().Err(errors.WithStack(err)).Msg("Error while constructing the validation request")
				http.Error(w, publicErrorMessage, http.StatusInternalServerError)
			}

			req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
			if err != nil {
				log.Error().Err(errors.WithStack(err)).Msg("Error creating NewRequestWithContext")
				http.Error(w, publicErrorMessage, http.StatusInternalServerError)
			}

			httpClient, err := rest.HTTPClientFor(k.restConfig)
			if err != nil {
				log.Error().Err(errors.WithStack(err)).Msg("Error creating httpClient")
				http.Error(w, publicErrorMessage, http.StatusInternalServerError)
			}
			httpClient.Transport = &TokenRoundTripper{Token: token}

			resp, err := httpClient.Do(req)
			if err != nil {
				log.Error().Err(errors.WithStack(err)).Msg("Token validation request failed")
				http.Error(w, publicErrorMessage, http.StatusInternalServerError)
			}
			defer func(Body io.ReadCloser) {
				err := Body.Close()
				if err != nil {
					log.Error().Err(errors.WithStack(err)).Msg("Error closing response body")
				}
			}(resp.Body)

			log.Debug().Int("status", resp.StatusCode).Msg("Token validation response received")

			// Check response status
			switch resp.StatusCode {
			case http.StatusUnauthorized:
				log.Debug().Msg("Token validation failed - unauthorized")
				http.Error(w, "invalid token", http.StatusUnauthorized)
			case http.StatusOK, http.StatusForbidden:
				// 200 OK means the token is valid and has access
				// 403 Forbidden means the token is valid but doesn't have permission (still authenticated)
				log.Debug().Int("status", resp.StatusCode).Msg("Token validation successful")
				next.ServeHTTP(w, r.WithContext(ctx))
			default:
				// Other status codes indicate an issue with the request or cluster
				log.Debug().Int("status", resp.StatusCode).Msg("Token validation failed with unexpected status")
				http.Error(w, publicErrorMessage, http.StatusInternalServerError)
			}

		})
	}
}

type TokenRoundTripper struct {
	Token     string
	Transport http.RoundTripper
}

func (trt *TokenRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	req2 := req.Clone(req.Context())
	req2.Header.Set("Authorization", "Bearer "+trt.Token)
	return trt.transport().RoundTrip(req2)
}

func (trt *TokenRoundTripper) transport() http.RoundTripper {
	if trt.Transport != nil {
		return trt.Transport
	}
	return http.DefaultTransport
}
