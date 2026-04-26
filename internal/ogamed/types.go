// Package ogamed provides a type-safe REST client for the ogamed daemon
// with rate limiting, retry logic, and envelope validation.
package ogamed

import "fmt"

// OgamedResponse is the standard envelope for ALL ogamed REST responses.
// Every endpoint returns: {"Status":"ok"|"error","Code":200,"Message":"","Result":...}
type OgamedResponse[T any] struct {
	Status  string `json:"Status"`
	Code    int    `json:"Code"`
	Message string `json:"Message"`
	Result  T      `json:"Result"`
}

// OgamedError represents an error response from ogamed.
type OgamedError struct {
	Message string
	Code    int
}

func (e *OgamedError) Error() string {
	return fmt.Sprintf("ogamed error %d: %s", e.Code, e.Message)
}
