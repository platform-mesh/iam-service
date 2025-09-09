package middleware

import (
	"net/http"
	"strings"

	"github.com/go-http-utils/headers"
	"github.com/go-jose/go-jose/v4"
	"github.com/platform-mesh/golang-commons/context"
)

const tokenAuthPrefix = "BEARER"

var SignatureAlgorithms = []jose.SignatureAlgorithm{jose.RS256}

// StoreWebToken retrieves the actual JWT Token within the Authorization header, and it stores it in the context as a struct
func StoreWebToken() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
			ctx := request.Context()
			auth := strings.Split(request.Header.Get(headers.Authorization), " ")
			if len(auth) > 1 && strings.ToUpper(auth[0]) == tokenAuthPrefix {
				ctx = context.AddWebTokenToContext(ctx, auth[1], SignatureAlgorithms)
			}

			next.ServeHTTP(responseWriter, request.WithContext(ctx))
		})
	}
}
