package middleware

import (
	"net/http"
)

func CreateMiddlewares() []func(http.Handler) http.Handler {
	mws := make([]func(http.Handler) http.Handler, 0, 5)

	mws = append(mws, StoreWebToken())
	mws = append(mws, StoreAuthHeader())
	mws = append(mws, StoreSpiffeHeader())

	return mws
}
