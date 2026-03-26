package downloader

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"darfin/internal/models"
)

// YtDlp is a helper for dealing with yt-dlp execution
type YtDlp struct {
	configDir string
	exePath   string
	mu        sync.Mutex
}

// NewYtDlp creates a new YtDlp helper
func NewYtDlp(configDir string) *YtDlp {
	return &YtDlp{
		configDir: configDir,
		exePath:   filepath.Join(configDir, "ext", "yt-dlp.exe"),
	}
}

// IsSupported checks if the URL should be handled by yt-dlp
func IsSupported(url string) bool {
	u := strings.ToLower(url)
	return strings.Contains(u, "youtube.com") || 
	       strings.Contains(u, "youtu.be") || 
		   strings.HasSuffix(u, ".m3u8") || 
		   strings.HasSuffix(u, ".mpd")
}

// EnsureExecutable checks if yt-dlp.exe exists and downloads it if not
func (y *YtDlp) EnsureExecutable() error {
	y.mu.Lock()
	defer y.mu.Unlock()

	// Check if already exists
	if _, err := os.Stat(y.exePath); err == nil {
		return nil
	}

	// Create directory
	extDir := filepath.Dir(y.exePath)
	if err := os.MkdirAll(extDir, 0755); err != nil {
		return fmt.Errorf("failed to create ext dir: %w", err)
	}

	// Download yt-dlp.exe from GitHub
	fmt.Println("Downloading yt-dlp.exe (One-time setup)...")
	resp, err := http.Get("https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp.exe")
	if err != nil {
		return fmt.Errorf("failed to download yt-dlp: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status finding yt-dlp: %d", resp.StatusCode)
	}

	out, err := os.Create(y.exePath)
	if err != nil {
		return fmt.Errorf("failed to create yt-dlp file: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, resp.Body); err != nil {
		os.Remove(y.exePath)
		return fmt.Errorf("failed to write yt-dlp.exe: %w", err)
	}

	// Make executable to be safe (though windows doesn't strictly need it, good for WSL compat)
	os.Chmod(y.exePath, 0755)
	
	return nil
}

// Probe returns the filename and total size using yt-dlp -j
func (y *YtDlp) Probe(url string) (fileName string, totalSize int64, err error) {
	if err := y.EnsureExecutable(); err != nil {
		return "", 0, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, y.exePath, "--dump-json", "--no-playlist", url)
	var outBuf bytes.Buffer
	cmd.Stdout = &outBuf

	if err := cmd.Run(); err != nil {
		return "", 0, fmt.Errorf("yt-dlp probe failed: %w", err)
	}

	var data struct {
		Title       string  `json:"title"`
		Ext         string  `json:"ext"`
		Duration    float64 `json:"duration"`
		Filesize    int64   `json:"filesize"`
		FilesizeApp int64   `json:"filesize_approx"`
		Filename    string  `json:"_filename"`
	}

	if err := json.Unmarshal(outBuf.Bytes(), &data); err != nil {
		return "", 0, fmt.Errorf("failed to parse yt-dlp output: %w", err)
	}

	fileName = data.Title + "." + data.Ext
	// Fallback to _filename if available and safe
	if data.Filename != "" {
		fileName = filepath.Base(data.Filename)
	}
	
	// Sanitize filename
	fileName = strings.ReplaceAll(fileName, "/", "_")
	fileName = strings.ReplaceAll(fileName, "\\", "_")
	fileName = strings.ReplaceAll(fileName, ":", "_")
	fileName = strings.ReplaceAll(fileName, "*", "_")
	fileName = strings.ReplaceAll(fileName, "?", "_")
	fileName = strings.ReplaceAll(fileName, "\"", "_")
	fileName = strings.ReplaceAll(fileName, "<", "_")
	fileName = strings.ReplaceAll(fileName, ">", "_")
	fileName = strings.ReplaceAll(fileName, "|", "_")

	totalSize = data.Filesize
	if totalSize == 0 {
		totalSize = data.FilesizeApp
	}

	return fileName, totalSize, nil
}

// StartDownload executes yt-dlp to download the file and reports progress
func (y *YtDlp) StartDownload(ctx context.Context, item *models.DownloadItem, cb ProgressCallback) error {
	if err := y.EnsureExecutable(); err != nil {
		return err
	}

	// We pass safe path directly. yt-dlp handles combining audio/video if ffmpeg is in PATH
	savePath := item.SavePath

	// Build progress template for easy parsing
	// Example parsed: [download]  15.5% of 50.00MiB at 10.00MiB/s ETA 00:05
	args := []string{
		"--newline",
		"--no-playlist",
		"--progress-template", "download:[progress] %(progress._percent_str)s %(progress._total_bytes_estimate_str)s %(progress._speed_str)s",
		"-o", savePath,
	}

	// We shouldn't redownload fragments if they exist, yt-dlp handles resume by default 
	// (though it might just overwrite if we don't pass -c, but -c is default)
	
	args = append(args, item.URL)

	cmd := exec.CommandContext(ctx, y.exePath, args...)
	
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start yt-dlp: %w", err)
	}

	scanDone := make(chan struct{})

	// Track progress manually since yt-dlp does the actual file writing
	go func() {
		defer close(scanDone)
		scanner := bufio.NewScanner(stdoutPipe)
		// Regex to parse our custom progress template
		// [progress] 15.5% 50.00MiB 10.00MiB/s
		re := regexp.MustCompile(`\[progress\]\s+([\d\.]+)%\s+([^\s]+)\s+([^\s]+)`)
		
		var lastProgress float64
		for scanner.Scan() {
			line := scanner.Text()
			
			if strings.HasPrefix(line, "[progress]") {
				matches := re.FindStringSubmatch(line)
				if len(matches) >= 4 {
					percentStr := matches[1]
					speedStr := matches[3] // e.g., "1.50MiB/s"
					
					percent, _ := strconv.ParseFloat(percentStr, 64)
					
					var speed int64 = 0
					if strings.Contains(speedStr, "KiB/s") {
						val, _ := strconv.ParseFloat(strings.TrimSuffix(speedStr, "KiB/s"), 64)
						speed = int64(val * 1024)
					} else if strings.Contains(speedStr, "MiB/s") {
						val, _ := strconv.ParseFloat(strings.TrimSuffix(speedStr, "MiB/s"), 64)
						speed = int64(val * 1024 * 1024)
					}
					
					// Calculate mocked downloaded size from total
					total := item.TotalSize
					if total <= 0 {
						total = 100 * 1024 * 1024 // fallback 100MB for math
					}
					downloadedSize := int64((percent / 100) * float64(total))

					if percent != lastProgress {
						lastProgress = percent
						
						item.Lock()
						item.DownloadedSize = downloadedSize
						item.Progress = percent
						item.Speed = speed
						item.Unlock()

						if cb != nil {
							cb(models.DownloadProgress{
								ID:             item.ID,
								DownloadedSize: downloadedSize,
								TotalSize:      item.TotalSize,
								Speed:          speed,
								Progress:       percent,
								Status:         models.StatusDownloading,
							})
						}
					}
				}
			}
		}
	}()

	<-scanDone // Wait for reading to finish

	err = cmd.Wait()
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("yt-dlp error: %w", err)
	}

	return nil
}
