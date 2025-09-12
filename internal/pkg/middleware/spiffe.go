package middleware

import (
	"net/http"

	"github.com/platform-mesh/golang-commons/context"
	"github.com/platform-mesh/golang-commons/jwt"
)

func StoreSpiffeHeader() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
			ctx := request.Context()
			uriVal := jwt.GetSpiffeUrlValue(request.Header)

			if uriVal != nil {
				ctx = context.AddSpiffeToContext(ctx, *uriVal)
			}
			next.ServeHTTP(responseWriter, request.WithContext(ctx))
		})
	}
}
