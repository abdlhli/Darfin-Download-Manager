package downloader

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"darfin/internal/models"
)

// ProgressCallback is called with download progress updates
type ProgressCallback func(progress models.DownloadProgress)

const defaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36 DARFIN/1.0"

// setHeaders sets common headers for download requests
func setHeaders(req *http.Request) {
	req.Header.Set("User-Agent", defaultUserAgent)
	req.Header.Set("Accept", "*/*")
}

// Engine handles multi-threaded segmented downloads
type Engine struct {
	client          *http.Client
	defaultThreads  int
	tempDir         string
	progressCb      ProgressCallback
	speedLimiter    *SpeedLimiter
}

// NewEngine creates a new download engine
func NewEngine(tempDir string, defaultThreads int, cb ProgressCallback) *Engine {
	return &Engine{
		client: &http.Client{
			Timeout: 0, // no timeout for downloads
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return fmt.Errorf("stopped after 10 redirects")
				}
				setHeaders(req)
				return nil
			},
		},
		defaultThreads: defaultThreads,
		tempDir:        tempDir,
		progressCb:     cb,
	}
}

// SetSpeedLimiter sets a global speed limiter
func (e *Engine) SetSpeedLimiter(limiter *SpeedLimiter) {
	e.speedLimiter = limiter
}

// ProbeURL sends a HEAD request to determine file info and resume support
func (e *Engine) ProbeURL(url string) (fileName string, totalSize int64, resumable bool, err error) {
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return "", 0, false, err
	}
	setHeaders(req)

	resp, err := e.client.Do(req)
	if err != nil || resp.StatusCode >= 400 {
		// Try GET with range 0-0 as fallback
		req, _ = http.NewRequest("GET", url, nil)
		setHeaders(req)
		req.Header.Set("Range", "bytes=0-0")
		resp, err = e.client.Do(req)
		if err != nil {
			return "", 0, false, fmt.Errorf("failed to probe URL: %w", err)
		}
	}
	defer resp.Body.Close()

	// Get file name from Content-Disposition or URL
	fileName = extractFileName(resp, url)

	// Get content length
	totalSize = resp.ContentLength
	if totalSize <= 0 {
		// For range responses, calculate from Content-Range
		if cr := resp.Header.Get("Content-Range"); cr != "" {
			totalSize = parseContentRangeTotal(cr)
		}
	}

	// Check if server supports range requests
	resumable = resp.Header.Get("Accept-Ranges") == "bytes" ||
		resp.StatusCode == http.StatusPartialContent

	return fileName, totalSize, resumable, nil
}

// StartDownload begins downloading a file with multiple threads
func (e *Engine) StartDownload(ctx context.Context, item *models.DownloadItem) error {
	threadCount := item.ThreadCount
	if threadCount <= 0 {
		threadCount = e.defaultThreads
	}

	// Create temp directory for this download
	downloadTempDir := filepath.Join(e.tempDir, item.ID)
	if err := os.MkdirAll(downloadTempDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}

	// Ensure save directory exists
	saveDir := filepath.Dir(item.SavePath)
	if err := os.MkdirAll(saveDir, 0755); err != nil {
		return fmt.Errorf("failed to create save dir: %w", err)
	}

	// If file is small or server doesn't support ranges, single thread
	if item.TotalSize <= 0 || !item.Resumable {
		threadCount = 1
	}

	// Initialize segments if not already set (resume case)
	if len(item.Segments) == 0 {
		item.Segments = e.createSegments(item.TotalSize, threadCount, downloadTempDir)
	}

	// Track total downloaded bytes atomically
	var totalDownloaded int64 = 0

	// Calculate already downloaded bytes (for resume)
	for _, seg := range item.Segments {
		if seg.Completed {
			atomic.AddInt64(&totalDownloaded, seg.EndByte-seg.StartByte+1)
		} else {
			atomic.AddInt64(&totalDownloaded, seg.DownloadedBytes)
		}
	}

	// Speed tracker
	speedTracker := models.NewSpeedTracker()

	// Start progress reporter
	progressDone := make(chan struct{})
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				downloaded := atomic.LoadInt64(&totalDownloaded)
				speed := speedTracker.Update(downloaded)
				progress := float64(0)
				if item.TotalSize > 0 {
					progress = float64(downloaded) / float64(item.TotalSize) * 100
				}

				item.Lock()
				item.DownloadedSize = downloaded
				item.Speed = speed
				item.Progress = progress
				item.Unlock()

				if e.progressCb != nil {
					e.progressCb(models.DownloadProgress{
						ID:             item.ID,
						DownloadedSize: downloaded,
						TotalSize:      item.TotalSize,
						Speed:          speed,
						Progress:       progress,
						Status:         models.StatusDownloading,
					})
				}
			case <-progressDone:
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	// Download segments in parallel
	var wg sync.WaitGroup
	errChan := make(chan error, len(item.Segments))

	for i := range item.Segments {
		seg := &item.Segments[i]
		if seg.Completed {
			continue
		}

		wg.Add(1)
		go func(segment *models.Segment) {
			defer wg.Done()
			if err := e.downloadSegment(ctx, item.URL, segment, &totalDownloaded); err != nil {
				if ctx.Err() != nil {
					return // context cancelled (pause/cancel)
				}
				errChan <- fmt.Errorf("segment %d failed: %w", segment.Index, err)
			}
		}(seg)
	}

	wg.Wait()
	close(progressDone)
	close(errChan)

	// Check for context cancellation (pause/cancel)
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// Check for segment errors
	var firstErr error
	for err := range errChan {
		if firstErr == nil {
			firstErr = err
		}
	}
	if firstErr != nil {
		return firstErr
	}

	// Merge segments into final file
	item.Lock()
	item.Status = models.StatusMerging
	item.Unlock()

	if e.progressCb != nil {
		e.progressCb(models.DownloadProgress{
			ID:     item.ID,
			Status: models.StatusMerging,
		})
	}

	if err := MergeSegments(item.Segments, item.SavePath); err != nil {
		return fmt.Errorf("failed to merge segments: %w", err)
	}

	// Cleanup temp directory
	os.RemoveAll(downloadTempDir)

	return nil
}

// createSegments divides the file into segments
func (e *Engine) createSegments(totalSize int64, threadCount int, tempDir string) []models.Segment {
	if totalSize <= 0 {
		// Unknown size, single segment
		return []models.Segment{
			{
				Index:        0,
				StartByte:    0,
				EndByte:      -1, // unknown
				TempFilePath: filepath.Join(tempDir, "segment_0.tmp"),
			},
		}
	}

	segments := make([]models.Segment, threadCount)
	segmentSize := totalSize / int64(threadCount)

	for i := 0; i < threadCount; i++ {
		start := int64(i) * segmentSize
		end := start + segmentSize - 1
		if i == threadCount-1 {
			end = totalSize - 1
		}

		segments[i] = models.Segment{
			Index:        i,
			StartByte:    start,
			EndByte:      end,
			TempFilePath: filepath.Join(tempDir, fmt.Sprintf("segment_%d.tmp", i)),
		}
	}

	return segments
}

// downloadSegment downloads a single segment with Range request
func (e *Engine) downloadSegment(ctx context.Context, url string, segment *models.Segment, totalDownloaded *int64) error {
	// Calculate actual start (for resume)
	actualStart := segment.StartByte + segment.DownloadedBytes

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	setHeaders(req)

	// Set range header
	if segment.EndByte > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", actualStart, segment.EndByte))
	} else if actualStart > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", actualStart))
	}

	resp, err := e.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Open temp file for writing (append if resuming)
	flags := os.O_CREATE | os.O_WRONLY
	if segment.DownloadedBytes > 0 {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}

	file, err := os.OpenFile(segment.TempFilePath, flags, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// Read and write in chunks
	buf := make([]byte, 32*1024) // 32KB buffer
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		var reader io.Reader = resp.Body
		if e.speedLimiter != nil {
			reader = e.speedLimiter.Reader(resp.Body)
		}

		n, err := reader.Read(buf)
		if n > 0 {
			if _, writeErr := file.Write(buf[:n]); writeErr != nil {
				return writeErr
			}
			segment.DownloadedBytes += int64(n)
			atomic.AddInt64(totalDownloaded, int64(n))
		}

		if err != nil {
			if err == io.EOF {
				segment.Completed = true
				return nil
			}
			return err
		}
	}
}
