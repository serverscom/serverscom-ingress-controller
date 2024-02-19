package config

import (
	"fmt"

	serverscom "github.com/serverscom/serverscom-go-client/pkg"
)

const (
	DefaultSCApiUrl = "https://api.servers.com/v1"
)

// NewServerscomClient creates a new SC client to interact with SC public api
func NewServerscomClient() (*serverscom.Client, error) {
	token := FetchEnv("SC_ACCESS_TOKEN")
	apiUrl := FetchEnv("SC_API_URL", DefaultSCApiUrl)
	if apiUrl == "" {
		apiUrl = DefaultSCApiUrl
	}
	if token == "" {
		return nil, fmt.Errorf("SC_ACCESS_TOKEN env is empty, can't create SC client")
	}
	return serverscom.NewClientWithEndpoint(token, apiUrl), nil
}
