package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/openmfp/iam-service/pkg/graph"
	"github.com/openmfp/iam-service/pkg/resolver"
)

const defaultPort = "8080"

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	srv := handler.NewDefaultServer(graph.NewExecutableSchema(graph.Config{Resolvers: &resolver.Resolver{}}))

	http.Handle("/", playground.Handler("GraphQL playground", "/query"))
	http.Handle("/query", srv)

	log.Printf("connect to http://localhost:%s/ for GraphQL playground", port)

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      nil,
		ReadTimeout:  5 * time.Second,  // Set a reasonable read timeout
		WriteTimeout: 10 * time.Second, // Set a reasonable write timeout
		IdleTimeout:  15 * time.Second, // Set a reasonable idle timeout
	}

	log.Fatal(server.ListenAndServe())
}
