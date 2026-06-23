package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
	"golang.org/x/mod/semver"
)

// DependencyInfo describes a single Go module dependency, its currently used
// version and whether a newer version is available upstream.
type DependencyInfo struct {
	Path           string `json:"path"`
	CurrentVersion string `json:"current_version"`
	LatestVersion  string `json:"latest_version,omitempty"`
	Updatable      bool   `json:"updatable"`
	Indirect       bool   `json:"indirect"`
	Error          string `json:"error,omitempty"`
}

// Dependencies inspects the module dependencies declared in go.mod and queries
// the Go module proxy to determine which ones can be upgraded.
type Dependencies struct {
	// GoModPath is the absolute or relative path to the go.mod file to inspect.
	GoModPath string
	// ProxyBaseURL is the base URL of the Go module proxy used to look up the
	// latest available version for each module. Defaults to the public proxy.
	ProxyBaseURL string
	// HTTPClient is used to query the module proxy.
	HTTPClient *http.Client
}

// NewDependenciesService builds a Dependencies service. If goModPath is empty it
// defaults to "go.mod" in the current working directory. The GOPROXY environment
// variable (when set to a single direct URL) overrides the default proxy.
func NewDependenciesService(goModPath string) *Dependencies {
	if goModPath == "" {
		goModPath = "go.mod"
	}

	proxy := "https://proxy.golang.org"
	if env := os.Getenv("GOPROXY"); env != "" && env != "off" && env != "direct" {
		proxy = env
	}

	return &Dependencies{
		GoModPath:    goModPath,
		ProxyBaseURL: proxy,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// proxyVersionInfo is the response shape of the Go module proxy @latest endpoint.
type proxyVersionInfo struct {
	Version string `json:"Version"`
}

// List parses go.mod and returns every required dependency together with the
// latest version available on the configured proxy and an updatable flag.
func (d *Dependencies) List(ctx context.Context) ([]DependencyInfo, error) {
	data, err := os.ReadFile(d.GoModPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read go.mod at %q: %w", d.GoModPath, err)
	}

	mf, err := modfile.Parse(filepath.Base(d.GoModPath), data, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to parse go.mod: %w", err)
	}

	deps := make([]DependencyInfo, len(mf.Require))
	var wg sync.WaitGroup

	for i, req := range mf.Require {
		i, req := i, req
		deps[i] = DependencyInfo{
			Path:           req.Mod.Path,
			CurrentVersion: req.Mod.Version,
			Indirect:       req.Indirect,
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			latest, err := d.latestVersion(ctx, req.Mod.Path)
			if err != nil {
				deps[i].Error = err.Error()
				return
			}
			deps[i].LatestVersion = latest
			if semver.IsValid(req.Mod.Version) && semver.IsValid(latest) {
				deps[i].Updatable = semver.Compare(latest, req.Mod.Version) > 0
			}
		}()
	}

	wg.Wait()

	sort.Slice(deps, func(a, b int) bool {
		return deps[a].Path < deps[b].Path
	})

	return deps, nil
}

// UpdateResult describes the outcome of upgrading a single dependency.
type UpdateResult struct {
	Path            string `json:"path"`
	PreviousVersion string `json:"previous_version"`
	NewVersion      string `json:"new_version"`
}

// Update upgrades the given module to the requested version (or to its latest
// available version when targetVersion is empty) by invoking the Go toolchain
// against the module that owns the configured go.mod file. It returns the
// dependency information after the update has been applied.
//
// The update is performed with `go get <module>@<version>` followed by
// `go mod tidy` so that go.mod and go.sum stay consistent. Both commands run in
// the directory containing the configured go.mod file.
func (d *Dependencies) Update(ctx context.Context, modulePath, targetVersion string) (*UpdateResult, error) {
	modulePath = strings.TrimSpace(modulePath)
	if modulePath == "" {
		return nil, fmt.Errorf("module path is required")
	}
	if err := module.CheckPath(modulePath); err != nil {
		return nil, fmt.Errorf("invalid module path %q: %w", modulePath, err)
	}

	// Resolve the dependency to ensure it is actually declared and to capture
	// its current version before applying the update.
	current, err := d.requirement(modulePath)
	if err != nil {
		return nil, err
	}

	targetVersion = strings.TrimSpace(targetVersion)
	if targetVersion == "" {
		latest, err := d.latestVersion(ctx, modulePath)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve latest version: %w", err)
		}
		targetVersion = latest
	} else if !semver.IsValid(targetVersion) {
		return nil, fmt.Errorf("invalid target version %q", targetVersion)
	}

	if current.Version != "" && semver.IsValid(current.Version) && semver.IsValid(targetVersion) &&
		semver.Compare(targetVersion, current.Version) < 0 {
		return nil, fmt.Errorf("target version %s is older than current version %s", targetVersion, current.Version)
	}

	workDir := filepath.Dir(d.GoModPath)
	if workDir == "" {
		workDir = "."
	}

	if out, err := d.runGo(ctx, workDir, "get", modulePath+"@"+targetVersion); err != nil {
		return nil, fmt.Errorf("go get failed: %w: %s", err, strings.TrimSpace(out))
	}
	if out, err := d.runGo(ctx, workDir, "mod", "tidy"); err != nil {
		return nil, fmt.Errorf("go mod tidy failed: %w: %s", err, strings.TrimSpace(out))
	}

	// Re-read go.mod to report the version that was actually pinned.
	updated, err := d.requirement(modulePath)
	if err != nil {
		return nil, err
	}

	return &UpdateResult{
		Path:            modulePath,
		PreviousVersion: current.Version,
		NewVersion:      updated.Version,
	}, nil
}

// requirement parses go.mod and returns the require directive for the given
// module path.
func (d *Dependencies) requirement(modulePath string) (module.Version, error) {
	data, err := os.ReadFile(d.GoModPath)
	if err != nil {
		return module.Version{}, fmt.Errorf("failed to read go.mod at %q: %w", d.GoModPath, err)
	}

	mf, err := modfile.Parse(filepath.Base(d.GoModPath), data, nil)
	if err != nil {
		return module.Version{}, fmt.Errorf("failed to parse go.mod: %w", err)
	}

	for _, req := range mf.Require {
		if req.Mod.Path == modulePath {
			return req.Mod, nil
		}
	}

	return module.Version{}, fmt.Errorf("dependency %q is not declared in go.mod", modulePath)
}

// runGo executes a `go` subcommand in the given working directory and returns
// its combined output.
func (d *Dependencies) runGo(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOFLAGS=-mod=mod")
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// latestVersion queries the module proxy for the latest version of the given
// module path.
func (d *Dependencies) latestVersion(ctx context.Context, modulePath string) (string, error) {
	escaped, err := module.EscapePath(modulePath)
	if err != nil {
		return "", fmt.Errorf("invalid module path: %w", err)
	}

	endpoint := fmt.Sprintf("%s/%s/@latest", d.ProxyBaseURL, escaped)
	if _, err := url.Parse(endpoint); err != nil {
		return "", fmt.Errorf("invalid proxy url: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}

	resp, err := d.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("proxy returned status %d", resp.StatusCode)
	}

	var info proxyVersionInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return "", fmt.Errorf("failed to decode proxy response: %w", err)
	}

	return info.Version, nil
}
