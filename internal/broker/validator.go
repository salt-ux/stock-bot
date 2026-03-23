package broker

import "context"

type ValidationResult struct {
	Valid      bool   `json:"valid"`
	StatusCode int    `json:"status_code,omitempty"`
	Message    string `json:"message"`
}

type CredentialValidator interface {
	ValidateCredentials(ctx context.Context) ValidationResult
}
