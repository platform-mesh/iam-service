package gqlAssertions

import (
	"errors"
	"fmt"
	"net/http"

	jsonPathInt "github.com/steinfletcher/apitest-jsonpath/jsonpath"
)

func NoGQLErrors() func(response *http.Response, request *http.Request) error {
	return func(response *http.Response, request *http.Request) error {
		expression := "$.errors"
		value, _ := jsonPathInt.JsonPath(response.Body, expression)
		if value != nil {
			return fmt.Errorf("errors in graphql: '%s', '%s'", expression, value)
		}
		return nil
	}
}

func HasGQLErrors() func(response *http.Response, request *http.Request) error {
	return func(response *http.Response, request *http.Request) error {
		expression := "$.errors"
		value, _ := jsonPathInt.JsonPath(response.Body, expression)
		if value == nil {
			return errors.New("expected errors in graphql, but got none")
		}
		return nil
	}
}
