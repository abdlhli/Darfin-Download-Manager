package downloader

import (
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"
)

// extractFileName extracts the filename from HTTP response or URL
func extractFileName(resp *http.Response, rawURL string) string {
	// Try Content-Disposition header first
	if cd := resp.Header.Get("Content-Disposition"); cd != "" {
		_, params, err := mime.ParseMediaType(cd)
		if err == nil {
			if name, ok := params["filename"]; ok && name != "" {
				return sanitizeFileName(name)
			}
		}
	}

	// Extract from URL path
	parsedURL, err := url.Parse(rawURL)
	if err == nil {
		urlPath := parsedURL.Path
		name := path.Base(urlPath)
		if name != "" && name != "." && name != "/" {
			// Decode URL encoding
			decoded, err := url.PathUnescape(name)
			if err == nil {
				name = decoded
			}
			return sanitizeFileName(name)
		}
	}

	// Fallback
	return "download"
}

// parseContentRangeTotal parses total size from Content-Range header
// Format: bytes 0-499/1234
func parseContentRangeTotal(cr string) int64 {
	parts := strings.Split(cr, "/")
	if len(parts) == 2 {
		total, err := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
		if err == nil {
			return total
		}
	}
	return -1
}

// sanitizeFileName removes invalid characters from filename
func sanitizeFileName(name string) string {
	// Remove invalid Windows filename characters
	re := regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)
	name = re.ReplaceAllString(name, "_")

	// Trim spaces and dots
	name = strings.TrimRight(name, " .")
	name = strings.TrimSpace(name)

	if name == "" {
		return "download"
	}

	return name
}

// FormatBytes formats bytes to human readable string
func FormatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// FormatSpeed formats bytes/sec to human readable string
func FormatSpeed(bytesPerSec int64) string {
	return FormatBytes(bytesPerSec) + "/s"
}
