package handlers

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	sdk "atonixcorp/cloud/devtools/sdk/go"
)

func newSDKClient() (*sdk.Client, error) {
	rawEndpoint := strings.TrimRight(os.Getenv("ATCLOUD_ENDPOINT"), "/")
	token := os.Getenv("ATCLOUD_TOKEN")
	endpoint, err := url.Parse(rawEndpoint)
	if err != nil || endpoint.Host == "" || (endpoint.Scheme != "http" && endpoint.Scheme != "https") || endpoint.User != nil {
		return nil, fmt.Errorf("ATCLOUD_ENDPOINT must be an absolute HTTP(S) URL")
	}

	if token == "" {
		return nil, fmt.Errorf("ATCLOUD_TOKEN is required")
	}
	return sdk.NewClient(sdk.ClientOptions{
		Endpoint: rawEndpoint,
		Token:    token,
		Timeout:  10 * time.Second,
	}), nil
}

func ValidateUpstreamConfiguration() (*sdk.Client, error) {
	return newSDKClient()
}
