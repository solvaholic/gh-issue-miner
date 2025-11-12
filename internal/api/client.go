package api

import (
	"strings"
	"time"

	ghapi "github.com/cli/go-gh/v2/pkg/api"
)

// RESTClient defines the subset of methods we use from go-gh's REST client.
type RESTClient interface {
	Get(path string, out interface{}) error
}

// retryClient wraps a RESTClient and retries transient failures with exponential backoff.
type retryClient struct {
	inner     RESTClient
	attempts  int
	baseDelay time.Duration
}

func (r *retryClient) Get(path string, out interface{}) error {
	var last error
	delay := r.baseDelay
	for i := 0; i < r.attempts; i++ {
		last = r.inner.Get(path, out)
		if last == nil {
			return nil
		}
		// conservative non-retriable detection: 404 / 401
		s := last.Error()
		if strings.Contains(s, "404") || strings.Contains(s, "Not Found") || strings.Contains(s, "401") || strings.Contains(s, "Unauthorized") {
			return last
		}
		time.Sleep(delay)
		delay *= 2
	}
	return last
}

// NewRESTClient returns the default go-gh REST client wrapped with retry/backoff.
func NewRESTClient() (RESTClient, error) {
	c, err := ghapi.DefaultRESTClient()
	if err != nil {
		return nil, err
	}
	return &retryClient{inner: c, attempts: 3, baseDelay: 500 * time.Millisecond}, nil
}

// NewClient is a variable wrapper around NewRESTClient so tests can override it.
var NewClient = NewRESTClient
