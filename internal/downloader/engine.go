package downloader

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"darfin/internal/models"
)

// ProgressCallback is called with download progress updates
type ProgressCallback func(progress models.DownloadProgress)

const defaultUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36 DARFIN/1.0"

// setHeaders sets common headers for download requests
func setHeaders(req *http.Request, cookies string, referrer string) {
	req.Header.Set("User-Agent", defaultUserAgent)
	req.Header.Set("Accept", "*/*")
	if cookies != "" {
		req.Header.Set("Cookie", cookies)
	}
	if referrer != "" {
		req.Header.Set("Referer", referrer)
	}
}

var bufferPool = sync.Pool{
	New: func() interface{} {
		buf := make([]byte, 128*1024)
		return &buf
	},
}

// Engine handles multi-threaded segmented downloads
type Engine struct {
	client         *http.Client
	defaultThreads int
	tempDir        string // Kept for interface compatibility
	progressCb     ProgressCallback
	speedLimiter   *SpeedLimiter
}

// NewEngine creates a new download engine
func NewEngine(tempDir string, defaultThreads int, cb ProgressCallback) *Engine {
	return &Engine{
		client: &http.Client{
			Timeout: 0, // no timeout for downloads
			Transport: &http.Transport{
				MaxIdleConns:        1000,
				MaxIdleConnsPerHost: 100,
				IdleConnTimeout:     90 * time.Second,
			},
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return fmt.Errorf("stopped after 10 redirects")
				}
				setHeaders(req, "", "")
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
func (e *Engine) ProbeURL(url string, cookies string, referrer string) (fileName string, totalSize int64, resumable bool, err error) {
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return "", 0, false, err
	}
	setHeaders(req, cookies, referrer)

	resp, err := e.client.Do(req)
	if err != nil || resp.StatusCode >= 400 {
		// Try GET with range 0-0 as fallback
		req, _ = http.NewRequest("GET", url, nil)
		setHeaders(req, cookies, referrer)
		req.Header.Set("Range", "bytes=0-0")
		resp, err = e.client.Do(req)
		if err != nil {
			return "", 0, false, fmt.Errorf("failed to probe URL: %w", err)
		}
	}
	defer resp.Body.Close()

	// Check if server returned a web page instead of a file
	contentType := resp.Header.Get("Content-Type")
	isHtmlCtype := strings.Contains(strings.ToLower(contentType), "text/html")
	isHtmlFile := strings.HasSuffix(strings.ToLower(url), ".html") || strings.HasSuffix(strings.ToLower(url), ".htm")

	if isHtmlCtype && !isHtmlFile {
		return "", 0, false, fmt.Errorf("server memblokir fitur unduh (mengembalikan halaman peringatan HTML / batas limit kuota / link kedaluwarsa)")
	}

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

	// Ensure save directory exists
	saveDir := filepath.Dir(item.SavePath)
	if err := os.MkdirAll(saveDir, 0755); err != nil {
		return fmt.Errorf("failed to create save dir: %w", err)
	}

	// If file is small or server doesn't support ranges, single thread
	if item.TotalSize <= 0 || !item.Resumable {
		threadCount = 1
	}

	// Output file path
	darfinPath := item.SavePath + ".darfin"
	fileMode := os.O_CREATE | os.O_WRONLY
	if item.TotalSize > 0 && item.DownloadedSize > 0 {
		fileMode = os.O_RDWR // Resuming
	} else if item.TotalSize > 0 {
		fileMode = os.O_CREATE | os.O_RDWR
	}

	dstFile, err := os.OpenFile(darfinPath, fileMode, 0644)
	if err != nil {
		return fmt.Errorf("failed to open destination file: %w", err)
	}
	defer dstFile.Close()

	if item.TotalSize > 0 && item.DownloadedSize == 0 {
		// Pre-allocate space
		if err := dstFile.Truncate(item.TotalSize); err != nil {
			return fmt.Errorf("failed to pre-allocate disk space: %w", err)
		}
	}

	item.Lock()
	// Initialize segments if not already set (resume case)
	if len(item.Segments) == 0 {
		item.Segments = e.createSegments(item.TotalSize, threadCount)
	} else {
		// Ensure capacity is 200 to prevent pointer shifts during work stealing
		if cap(item.Segments) < 200 {
			newSegs := make([]models.Segment, len(item.Segments), 200)
			copy(newSegs, item.Segments)
			item.Segments = newSegs
		}
	}
	item.Unlock()

	// Track total downloaded bytes atomically
	var totalDownloaded int64 = 0
	item.Lock()
	for _, seg := range item.Segments {
		if seg.Completed {
			atomic.AddInt64(&totalDownloaded, seg.EndByte-seg.StartByte+1)
		} else {
			atomic.AddInt64(&totalDownloaded, seg.DownloadedBytes)
		}
	}
	item.Unlock()

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
				if item.SpeedLimiter == nil {
					item.SpeedLimiter = NewSpeedLimiter(0) // Default untethered limiter
				}
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

	// Worker pool
	var wg sync.WaitGroup
	errChan := make(chan error, 1) // Only track first critical error

	// Channel to signal there's work
	workChan := make(chan int, 200)

	item.Lock()
	for i := range item.Segments {
		if !item.Segments[i].Completed {
			workChan <- i
		}
	}
	item.Unlock()

	// Start workers
	activeThreads := threadCount
	if activeThreads > len(workChan) {
		activeThreads = len(workChan)
	}
	if activeThreads == 0 {
		activeThreads = 1 // Just in case
	}

	for i := 0; i < activeThreads; i++ {
		wg.Add(1)
		workerID := i
		go func() {
			defer wg.Done()

			// STAGGER CONNECTION START TO PREVENT FIREWALLS/WAF FROM DROPPING CONNECTIONS
			if workerID > 0 {
				select {
				case <-ctx.Done():
					return
				case <-time.After(time.Duration(workerID) * 250 * time.Millisecond):
				}
			}

			for {
				select {
				case <-ctx.Done():
					return
				case segIdx := <-workChan:
					var err error
				retryLoop:
					for attempt := 1; attempt <= 5; attempt++ {
						err = e.downloadSegment(ctx, item, segIdx, dstFile, &totalDownloaded)
						if err == nil || err == context.Canceled {
							break retryLoop
						}
						// If error occurs, wait before retrying (linear backoff: 1s, 2s, 3s...)
						select {
						case <-ctx.Done():
							err = ctx.Err()
							break retryLoop
						case <-time.After(time.Duration(attempt) * time.Second):
							// Retry
						}
					}

					if err != nil && err != context.Canceled {
						select {
						case errChan <- fmt.Errorf("segment failed after retries: %w", err):
						default:
						}
						return
					}

					// Work Stealing
					if err == nil && item.Resumable {
						newSegIdx := e.stealWork(item)
						if newSegIdx >= 0 {
							workChan <- newSegIdx
						}
					}
				default:
					return // No more work in channel
				}
			}
		}()
	}

	wg.Wait()
	close(progressDone)

	if ctx.Err() != nil {
		return ctx.Err()
	}

	select {
	case err := <-errChan:
		return err
	default:
	}

	// Rename .darfin to final filename
	dstFile.Close() // Must close before renaming on Windows
	if err := os.Rename(darfinPath, item.SavePath); err != nil {
		return fmt.Errorf("failed to finalize file: %w", err)
	}

	return nil
}

// stealWork dynamically splits the largest active segment to balance load
func (e *Engine) stealWork(item *models.DownloadItem) int {
	item.Lock()
	defer item.Unlock()

	var largestSegIdx int = -1
	var maxRemaining int64 = 0

	for i := range item.Segments {
		seg := &item.Segments[i]
		if seg.Completed {
			continue
		}

		end := atomic.LoadInt64(&seg.EndByte)
		down := atomic.LoadInt64(&seg.DownloadedBytes)
		remaining := end - (seg.StartByte + down)

		if remaining > maxRemaining {
			maxRemaining = remaining
			largestSegIdx = i
		}
	}

	// Only steal if remaining > 2MB to prevent micro-segment fragmentation
	if maxRemaining > 2*1024*1024 {
		largestSeg := &item.Segments[largestSegIdx]

		end := atomic.LoadInt64(&largestSeg.EndByte)
		down := atomic.LoadInt64(&largestSeg.DownloadedBytes)
		currentPos := largestSeg.StartByte + down

		halfRemaining := maxRemaining / 2
		splitPoint := currentPos + halfRemaining

		newSeg := models.Segment{
			Index:           len(item.Segments),
			StartByte:       splitPoint,
			EndByte:         end,
			DownloadedBytes: 0,
			Completed:       false,
		}

		// Update original segment's EndByte atomically
		atomic.StoreInt64(&largestSeg.EndByte, splitPoint-1)

		item.Segments = append(item.Segments, newSeg)
		return newSeg.Index
	}

	return -1
}

// createSegments divides the file into segments
func (e *Engine) createSegments(totalSize int64, threadCount int) []models.Segment {
	segments := make([]models.Segment, threadCount, 200)

	if totalSize <= 0 {
		segments[0] = models.Segment{
			Index:     0,
			StartByte: 0,
			EndByte:   -1,
		}
		return segments[:1]
	}

	segmentSize := totalSize / int64(threadCount)

	for i := 0; i < threadCount; i++ {
		start := int64(i) * segmentSize
		end := start + segmentSize - 1
		if i == threadCount-1 {
			end = totalSize - 1
		}

		segments[i] = models.Segment{
			Index:     i,
			StartByte: start,
			EndByte:   end,
		}
	}

	return segments
}

// downloadSegment downloads a single segment with Range request directly to disk
func (e *Engine) downloadSegment(ctx context.Context, item *models.DownloadItem, segIdx int, dstFile *os.File, totalDownloaded *int64) error {
	item.Lock()
	segment := &item.Segments[segIdx]
	url := item.URL
	item.Unlock()

	actualStart := segment.StartByte + atomic.LoadInt64(&segment.DownloadedBytes)
	endByte := atomic.LoadInt64(&segment.EndByte)

	if endByte > 0 && actualStart > endByte {
		item.Lock()
		item.Segments[segIdx].Completed = true
		item.Unlock()
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	setHeaders(req, item.Cookies, item.Referrer)

	if endByte > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", actualStart, endByte))
	} else if actualStart > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", actualStart))
	}

	resp, err := e.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("unexpected status HTTP %d", resp.StatusCode)
	}

	bufPtr := bufferPool.Get().(*[]byte)
	buf := *bufPtr
	defer bufferPool.Put(bufPtr)

	const flushThreshold = 1024 * 1024 // 1MB buffer
	writeBuf := make([]byte, 0, flushThreshold+128*1024)
	writeOffset := actualStart

	flush := func() error {
		if len(writeBuf) > 0 && dstFile != nil {
			if _, writeErr := dstFile.WriteAt(writeBuf, writeOffset); writeErr != nil {
				return writeErr
			}
			writeOffset += int64(len(writeBuf))
			atomic.StoreInt64(&segment.DownloadedBytes, writeOffset-segment.StartByte)
			writeBuf = writeBuf[:0]
		}
		return nil
	}
	defer flush() // ensure anything left is written

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		currentEnd := atomic.LoadInt64(&segment.EndByte)
		currentOffset := writeOffset + int64(len(writeBuf))

		if currentEnd > 0 && currentOffset > currentEnd {
			break // Reached the dynamic end byte cleanly
		}

		toRead := int64(len(buf))
		if currentEnd > 0 {
			remain := currentEnd - currentOffset + 1
			if remain < toRead {
				toRead = remain
			}
		}

		var reader io.Reader = resp.Body
		if e.speedLimiter != nil {
			reader = e.speedLimiter.Reader(resp.Body)
		}

		item.Lock()
		dlLimiter := item.SpeedLimiter
		item.Unlock()

		if dlLimiter != nil {
			reader = dlLimiter.Reader(reader)
		}

		n, readErr := reader.Read(buf[:toRead])
		if n > 0 {
			// STRICT BOUNDARY CHECK: Re-evaluate currentEnd to prevent overlap if work was stolen during Read
			currentEnd = atomic.LoadInt64(&segment.EndByte)
			if currentEnd > 0 {
				maxAllowed := currentEnd - currentOffset + 1
				if maxAllowed <= 0 {
					n = 0 // Stolen entirely before we wrote
				} else if int64(n) > maxAllowed {
					n = int(maxAllowed) // Truncate overflow
				}
			}

			if n > 0 {
				writeBuf = append(writeBuf, buf[:n]...)
				atomic.AddInt64(totalDownloaded, int64(n)) // update visual tracker immediately

				if len(writeBuf) >= flushThreshold {
					if err := flush(); err != nil {
						return err
					}
				}
			}
		}

		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			return readErr
		}
	}

	item.Lock()
	item.Segments[segIdx].Completed = true
	item.Unlock()

	return flush()
}
