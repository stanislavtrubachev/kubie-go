package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
)

// VERSION is filled in when building via -ldflags,
// example: go build -ldflags "-X main.VERSION=1.2.3"
var VERSION string

const LATEST_RELEASE_URL string = "https://api.github.com/repos/sbstp/kubie/releases/latest"

// Release info
type Release struct {
	TagName string  `json:"tag_name"`
	Assets  []Asset `json:"assets"`
}

// GetLatest downloads information about the latest release from GitHub
func GetLatest() (*Release, error) {
	resp, err := http.Get(LATEST_RELEASE_URL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var rel Release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, err
	}
	return &rel, nil
}

// getBinaryName returns the name of the executable file for the current OS and architecture.
// Names correspond to the files in the releases GitHub: https://github.com/sbstp/kubie/releases.
func getBinaryName() (string, bool) {
	switch runtime.GOOS {
	case "linux":
		switch runtime.GOARCH {
		case "amd64":
			return "kubie-linux-amd64", true
		case "arm":
			return "kubie-linux-arm32", true
		case "arm64":
			return "kubie-linux-arm64", true
		}
	case "darwin":
		switch runtime.GOARCH {
		case "amd64":
			return "kubie-darwin-amd64", true
		case "arm64":
			return "kubie-darwin-arm64", true
		}
	}
	return "", false
}

// GetBinaryURL searches the list of release assets for a link to download a binary file.
func (r *Release) GetBinaryURL() (string, bool) {
	binaryName, ok := getBinaryName()
	if !ok {
		return "", false
	}
	for _, asset := range r.Assets {
		if strings.Contains(asset.BrowserDownloadURL, binaryName) {
			return asset.BrowserDownloadURL, true
		}
	}
	return "", false
}

type Asset struct {
	BrowserDownloadURL string `json:"browser_download_url"`
}

// Update checks for a new version of kubie-go on GitHub.
// If a new version is available, downloads a suitable binary for the current OS and architecture,
// replaces the current executable file with a new one.
// If successful, displays an update message, returns it if an error occurs.
// Requires an external VERSION constant and a ReplaceFile function.
func Update() error {
	latest, err := GetLatest()
	if err != nil {
		return err
	}

	if latest.TagName == "v"+VERSION {
		fmt.Printf("Kubie is up-to-date : v%s\n", VERSION)
		return nil
	}

	fmt.Printf("A new version of Kubie is available (%s), the new version will be installed by replacing this binary.\n", latest.TagName)

	downloadURL, ok := latest.GetBinaryURL()
	if !ok {
		return fmt.Errorf("Sorry, this release has no build for your OS, please create an issue : https://github.com/sbstp/kubie/issues")
	}

	fmt.Printf("Download url is: %s\n", downloadURL)

	resp, err := http.Get(downloadURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %s", resp.Status)
	}

	tmpFile, err := os.CreateTemp("", "kubie")
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}

	oldPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("Could not get own binary path: %w", err)
	}

	if err := replaceFile(oldPath, tmpFile.Name()); err != nil {
		return fmt.Errorf("Update failed. Consider using sudo?: %w", err)
	}

	fmt.Printf("Kubie has been updated successfully: %s\n", oldPath)
	return nil
}

// ReplaceFile replaces the old file with the contents of the new one.
// Sets rights 755 to the new file, deletes the old one, then copies the new one in its place.
func replaceFile(oldPath, newPath string) error {
	if err := os.Chmod(newPath, 0755); err != nil {
		return err
	}
	if err := os.Remove(oldPath); err != nil {
		return err
	}
	src, err := os.Open(newPath)
	if err != nil {
		return err
	}
	defer src.Close()
	dst, err := os.Create(oldPath)
	if err != nil {
		return err
	}
	defer dst.Close()
	if _, err := io.Copy(dst, src); err != nil {
		return err
	}
	return nil
}
