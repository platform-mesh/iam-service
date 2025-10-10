package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
	pmcontext "github.com/platform-mesh/golang-commons/context"
	"github.com/platform-mesh/golang-commons/logger"
	"github.com/rs/zerolog/log"
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

			// get tenantId from pmcontext
			clusterId, err := pmcontext.GetTenantFromContext(ctx)
			if err != nil {
				log.Error().Err(errors.WithStack(err)).Msg("Error while retrieving the tenant from the pmcontext")
				http.Error(w, "Error while retrieving the tenant from the pmcontext", http.StatusInternalServerError)
				return
			}

			// get token from pmcontext
			token, err := pmcontext.GetAuthHeaderFromContext(ctx)
			if err != nil {
				log.Error().Err(errors.WithStack(err)).Msg("Error while retrieving the token from the pmcontext")
				http.Error(w, "Error while retrieving the token from the pmcontext", http.StatusInternalServerError)
				return
			}

			clusterUrl, err := url.Parse(k.restConfig.Host)
			if err != nil {
				log.Error().Err(errors.WithStack(err)).Msg("Error parsing KCP host URL")
			}

			requestURL := fmt.Sprintf("%s://%s/clusters/%s/version", clusterUrl.Scheme, clusterUrl.Host, clusterId)

			req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, http.NoBody)
			if err != nil {
				http.Error(w, "Error validating token", http.StatusInternalServerError)
			}

			req.Header.Set("Authorization", "Bearer "+token)

			client, err := rest.HTTPClientFor(k.restConfig)
			if err != nil {
				http.Error(w, "Error creating client", http.StatusInternalServerError)
			}
			res, err := client.Do(req)
			if err != nil {
				http.Error(w, "Error validating token", http.StatusInternalServerError)
			}
			defer res.Body.Close() //nolint:errcheck

			switch res.StatusCode {
			case http.StatusOK, http.StatusCreated, http.StatusForbidden:
				// one could also continue here and use the OIDC userinfo endpoint to get more information about the user
				// but for now, just having a valid token is enough to be considered authenticated
				// even if the user does not have permissions to do anything (403)
				// this is similar to how the kube-apiserver handles authentication
				next.ServeHTTP(w, r.WithContext(ctx))

			default:
				http.Error(w, "invalid token", http.StatusUnauthorized)
			}
		})
	}
}

func (k *KCPValidation) validateTokenWithTokenReview(ctx context.Context, token string, clusterId string) (*TokenReviewStatus, error) {
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
		return nil, err
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
		return nil, err
	}

	// Set Content-Type header for JSON
	req.Header.Set("Content-Type", "application/json")

	httpClient, err := rest.HTTPClientFor(k.restConfig)
	if err != nil {
		log.Error().Err(errors.WithStack(err)).Msg("Error creating httpClient")
		return nil, err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		log.Error().Err(errors.WithStack(err)).Msg("TokenReview request failed")
		return nil, err
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
		return nil, err
	}

	// Parse the TokenReview response
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(errors.WithStack(err)).Msg("Error reading TokenReview response body")
		return nil, err
	}

	var tokenReviewResponse TokenReview
	if err := json.Unmarshal(responseBody, &tokenReviewResponse); err != nil {
		log.Error().Err(errors.WithStack(err)).Msg("Error unmarshaling TokenReview response")
		return nil, err
	}

	// Check if token is authenticated
	if !tokenReviewResponse.Status.Authenticated {
		if tokenReviewResponse.Status.Error != "" {
			log.Debug().Str("error", tokenReviewResponse.Status.Error).Msg("Token validation failed with error")
		} else {
			log.Debug().Msg("Token validation failed - not authenticated")
		}
		return nil, err
	}

	// Token is valid and authenticated
	log.Debug().Msg("Token validation successful")
	if tokenReviewResponse.Status.User != nil {
		log.Debug().Str("username", tokenReviewResponse.Status.User.Username).Msg("Authenticated user")
	}
	return &tokenReviewResponse.Status, nil
}
