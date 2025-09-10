package middleware

import (
	"net/http"

	"github.com/go-http-utils/headers"
	"github.com/platform-mesh/golang-commons/context"
)

// StoreAuthHeader stores the Authorization header within the request context
func StoreAuthHeader() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
			auth := request.Header.Get(headers.Authorization)
			ctx := context.AddAuthHeaderToContext(request.Context(), auth)
			next.ServeHTTP(responseWriter, request.WithContext(ctx))
		})
	}
}
