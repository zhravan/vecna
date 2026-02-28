package update

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
)

const (
	repoOwner = "zhravan"
	repoName  = "vecna"
	apiURL    = "https://api.github.com/repos/" + repoOwner + "/" + repoName + "/releases"
	releasesURL = "https://github.com/" + repoOwner + "/" + repoName + "/releases"
)

// FetchLatestVersion returns the latest release tag (e.g. "0.1.4") from GitHub.
func FetchLatestVersion() (string, error) {
	url := apiURL + "/latest"
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("releases/latest: %s", resp.Status)
	}
	var rel struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return "", err
	}
	// tag_name is e.g. "v0.1.4"
	tag := strings.TrimPrefix(rel.TagName, "v")
	if tag == "" {
		return "", fmt.Errorf("empty tag from GitHub")
	}
	return tag, nil
}

// DownloadAndReplace downloads the release for the current OS/arch, replaces the
// current binary, and exec's the new binary (never returns on success).
// version is e.g. "0.1.4" or "latest" (resolved via FetchLatestVersion).
func DownloadAndReplace(version string) error {
	if version == "latest" {
		var err error
		version, err = FetchLatestVersion()
		if err != nil {
			return fmt.Errorf("fetch latest: %w", err)
		}
	}
	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("executable path: %w", err)
	}
	self, err = filepath.EvalSymlinks(self)
	if err != nil {
		return fmt.Errorf("resolve symlink: %w", err)
	}
	dir := filepath.Dir(self)
	binName := filepath.Base(self)

	goos := runtime.GOOS
	goarch := runtime.GOARCH
	if goarch == "amd64" {
		goarch = "amd64"
	} else if goarch == "arm64" || goarch == "aarch64" {
		goarch = "arm64"
	} else {
		return fmt.Errorf("unsupported arch: %s", runtime.GOARCH)
	}
	if goos != "linux" && goos != "darwin" {
		return fmt.Errorf("in-app update only supported on linux and darwin (current: %s)", goos)
	}

	archiveName := fmt.Sprintf("vecna_%s_%s_%s.tar.gz", version, goos, goarch)
	url := fmt.Sprintf("%s/download/v%s/%s", releasesURL, version, archiveName)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download: %s", resp.Status)
	}
	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		return fmt.Errorf("gzip: %w", err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	var buf []byte
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar: %w", err)
		}
		if h.Typeflag != tar.TypeReg {
			continue
		}
		name := filepath.Base(h.Name)
		if name != "vecna" && name != binName {
			continue
		}
		buf, err = io.ReadAll(tr)
		if err != nil {
			return fmt.Errorf("read member: %w", err)
		}
		break
	}
	if len(buf) == 0 {
		return fmt.Errorf("no vecna binary in archive")
	}
	tmpPath := filepath.Join(dir, ".vecna.new."+version)
	if err := os.WriteFile(tmpPath, buf, 0755); err != nil {
		return fmt.Errorf("write temp: %w", err)
	}
	oldPath := self + ".old"
	_ = os.Remove(oldPath)
	if err := os.Rename(self, oldPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename current to .old: %w", err)
	}
	if err := os.Rename(tmpPath, self); err != nil {
		os.Rename(oldPath, self)
		return fmt.Errorf("rename new to current: %w", err)
	}
	// Replace process with new binary.
	args := os.Args
	env := os.Environ()
	if err := syscall.Exec(self, args, env); err != nil {
		return fmt.Errorf("exec new binary: %w", err)
	}
	return nil
}
