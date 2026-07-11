package clientbuild

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/eve-online-tools/eve-resfile-proxy/common/domain"
	"github.com/eve-online-tools/eve-resfile-proxy/common/platform"
)

type ClientBuild struct {
	// Both Build and BuildNumber give the same information, but it seems that `BuildNumber` is the way forward.
	Build       string `json:"build"`
	BuildNumber string `json:"buildNumber"`

	// Protected builds can only be downloaded with a valid SSO token.
	Protected bool `json:"protected"`

	// Additional platforms that the client build is available for. Windows is always available.
	Platforms []platform.Platform `json:"platforms"`
}

var (
	// Singularity and Tranquility seem to use a different name for the client build files.
	serverMap = map[string]string{
		"tranquility": "tq",
		"singularity": "sisi",
	}

	ErrInvalidRequest    = errors.New("invalid request")
	ErrInvalidStatusCode = errors.New("invalid status code")
	ErrInvalidResponse   = errors.New("invalid response")
)

func buildClientBuildURL(serverName string) string {
	name, ok := serverMap[serverName]
	if !ok {
		name = serverName
	}

	return domain.Binaries.URL(fmt.Sprintf("eveclient_%s.json", strings.ToUpper(name)))
}

func LoadClientBuild(ctx context.Context, client *http.Client, serverName string) (*ClientBuild, error) {
	url := buildClientBuildURL(serverName)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidRequest, err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() //nolint:errcheck // body fully read or discarded on error paths

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %d", ErrInvalidStatusCode, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidResponse, err)
	}

	var build ClientBuild
	if err := json.Unmarshal(body, &build); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidResponse, err)
	}

	// the `platforms` field never includes `windows`, because it is the default platform.
	build.Platforms = append(build.Platforms, platform.Windows)

	return &build, nil
}
