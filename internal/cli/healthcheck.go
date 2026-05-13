package cli

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

const (
	defaultHealthcheckPort    = "8080"
	defaultHealthcheckTimeout = 5 * time.Second
	healthcheckPath           = "/healthz"
)

// RunHealthcheckCommand probes the local server's /healthz endpoint and returns
// nil on a successful 2xx response. It is designed for use as a container
// healthcheck in scratch-based runtime images where no external HTTP client
// (curl/wget) is available.
func RunHealthcheckCommand(port string, timeout time.Duration) error {
	port = strings.TrimSpace(port)
	if port == "" {
		port = defaultHealthcheckPort
	}
	if timeout <= 0 {
		timeout = defaultHealthcheckTimeout
	}

	url := "http://127.0.0.1:" + port + healthcheckPath
	return probeHealthEndpoint(url, timeout)
}

func probeHealthEndpoint(url string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("build healthcheck request: %w", err)
	}

	client := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			DisableKeepAlives: true,
			DialContext: (&net.Dialer{
				Timeout: timeout,
			}).DialContext,
		},
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("healthcheck request failed: %w", err)
	}
	defer func() {
		_ = response.Body.Close()
	}()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("healthcheck returned status %d", response.StatusCode)
	}
	return nil
}
