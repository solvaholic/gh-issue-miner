package api

import (
	ghapi "github.com/cli/go-gh/v2/pkg/api"
)

// NewRESTClient returns the default go-gh REST client.
func NewRESTClient() (*ghapi.RESTClient, error) {
	return ghapi.DefaultRESTClient()
}
