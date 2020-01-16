package apiclient

import "fmt"

type APIError struct {
	Path         string
	StatusCode   int
	ResponseBody string
	Err          error
}

func NewAPIError(path, text string, code int, err error) *APIError {
	return &APIError{
		StatusCode:   code,
		ResponseBody: text,
		Err:          err,
		Path:         path,
	}
}

func (ae *APIError) Error() string {
	return fmt.Sprintf(
		"api error: %s -> %d %s: %s",
		ae.Path,
		ae.StatusCode,
		ae.ResponseBody,
		ae.Err,
	)
}

func (ae *APIError) Unwrap() error {
	return ae.Err
}
