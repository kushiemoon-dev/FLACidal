package backend

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// FFmpegInstallProgress represents progress during FFmpeg installation
type FFmpegInstallProgress struct {
	Stage   string  `json:"stage"`
	Percent float64 `json:"percent"`
	Error   string  `json:"error,omitempty"`
}

// Download URLs for static FFmpeg builds
var ffmpegDownloadURLs = map[string]string{
	"linux/amd64":   "https://johnvansickle.com/ffmpeg/releases/ffmpeg-release-amd64-static.tar.xz",
	"linux/arm64":   "https://johnvansickle.com/ffmpeg/releases/ffmpeg-release-arm64-static.tar.xz",
	"darwin/amd64":  "https://evermeet.cx/ffmpeg/getrelease/zip",
	"darwin/arm64":  "https://evermeet.cx/ffmpeg/getrelease/zip",
	"windows/amd64": "https://www.gyan.dev/ffmpeg/builds/ffmpeg-release-essentials.zip",
}

// GetFFmpegBinDir returns the path to ~/.flacidal/bin
func GetFFmpegBinDir() string {
	return filepath.Join(GetDataDir(), "bin")
}

// GetLocalFFmpegPath returns the path to the locally installed ffmpeg binary
func GetLocalFFmpegPath() string {
	name := "ffmpeg"
	if runtime.GOOS == "windows" {
		name = "ffmpeg.exe"
	}
	return filepath.Join(GetFFmpegBinDir(), name)
}

// IsFFmpegInstalledLocally checks if the local ffmpeg binary exists
func IsFFmpegInstalledLocally() bool {
	_, err := os.Stat(GetLocalFFmpegPath())
	return err == nil
}

// InstallFFmpeg downloads and installs a static FFmpeg binary
func InstallFFmpeg(progressCh chan<- FFmpegInstallProgress) error {
	defer close(progressCh)

	// Create bin directory
	binDir := GetFFmpegBinDir()
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return fmt.Errorf("failed to create bin directory: %w", err)
	}

	// Determine download URL
	key := runtime.GOOS + "/" + runtime.GOARCH
	url, ok := ffmpegDownloadURLs[key]
	if !ok {
		return fmt.Errorf("unsupported platform: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	// Download the archive
	progressCh <- FFmpegInstallProgress{Stage: "downloading", Percent: 0}

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download FFmpeg: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %s", resp.Status)
	}

	totalSize, _ := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)

	// Determine archive filename
	archiveExt := ".tar.xz"
	if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		archiveExt = ".zip"
	}
	archivePath := filepath.Join(binDir, "ffmpeg-download"+archiveExt)

	outFile, err := os.Create(archivePath)
	if err != nil {
		return fmt.Errorf("failed to create archive file: %w", err)
	}
	defer os.Remove(archivePath)

	// Download with progress tracking
	counter := &progressWriter{
		total:      totalSize,
		progressCh: progressCh,
	}
	_, err = io.Copy(outFile, io.TeeReader(resp.Body, counter))
	outFile.Close()
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	// Extract
	progressCh <- FFmpegInstallProgress{Stage: "extracting", Percent: 0}

	if archiveExt == ".tar.xz" {
		err = extractTarXz(archivePath, binDir)
	} else {
		err = extractZip(archivePath, binDir, runtime.GOOS)
	}
	if err != nil {
		return fmt.Errorf("extraction failed: %w", err)
	}

	// Make executable on Unix
	if runtime.GOOS != "windows" {
		if err := os.Chmod(GetLocalFFmpegPath(), 0755); err != nil {
			return fmt.Errorf("failed to set executable permission: %w", err)
		}
	}

	progressCh <- FFmpegInstallProgress{Stage: "complete", Percent: 100}
	return nil
}

// progressWriter tracks bytes written for download progress
type progressWriter struct {
	written    int64
	total      int64
	progressCh chan<- FFmpegInstallProgress
}

func (pw *progressWriter) Write(p []byte) (int, error) {
	n := len(p)
	pw.written += int64(n)
	if pw.total > 0 {
		pct := float64(pw.written) / float64(pw.total) * 100
		pw.progressCh <- FFmpegInstallProgress{Stage: "downloading", Percent: pct}
	}
	return n, nil
}

// extractTarXz extracts the ffmpeg binary from a tar.xz archive using the tar command
func extractTarXz(archivePath, destDir string) error {
	cmd := exec.Command("tar", "xJf", archivePath,
		"--strip-components=1", "-C", destDir, "--wildcards", "*/ffmpeg")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("tar extraction failed: %v - %s", err, string(output))
	}
	return nil
}

// extractZip extracts the ffmpeg binary from a zip archive
func extractZip(archivePath, destDir, goos string) error {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("failed to open zip: %w", err)
	}
	defer r.Close()

	targetName := "ffmpeg"
	if goos == "windows" {
		targetName = "ffmpeg.exe"
	}

	for _, f := range r.File {
		baseName := filepath.Base(f.Name)
		if !strings.EqualFold(baseName, targetName) {
			continue
		}

		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("failed to open zip entry: %w", err)
		}

		outPath := filepath.Join(destDir, targetName)
		outFile, err := os.Create(outPath)
		if err != nil {
			rc.Close()
			return fmt.Errorf("failed to create output file: %w", err)
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()
		if err != nil {
			return fmt.Errorf("failed to extract file: %w", err)
		}

		return nil
	}

	return fmt.Errorf("ffmpeg binary not found in zip archive")
}
