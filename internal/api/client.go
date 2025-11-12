package api

import (
	ghapi "github.com/cli/go-gh/v2/pkg/api"
)

// RESTClient defines the subset of methods we use from go-gh's REST client.
type RESTClient interface {
	Get(path string, out interface{}) error
}

// NewRESTClient returns the default go-gh REST client as a RESTClient interface.
func NewRESTClient() (RESTClient, error) {
	return ghapi.DefaultRESTClient()
}

// NewClient is a variable wrapper around NewRESTClient so tests can override it.
var NewClient = NewRESTClient
