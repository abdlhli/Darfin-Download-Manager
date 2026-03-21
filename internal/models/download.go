package models

import (
	"io"
	"sync"
	"time"
)

// DownloadStatus represents the current state of a download
type DownloadStatus string

const (
	StatusPending     DownloadStatus = "pending"
	StatusDownloading DownloadStatus = "downloading"
	StatusPaused      DownloadStatus = "paused"
	StatusCompleted   DownloadStatus = "completed"
	StatusError       DownloadStatus = "error"
	StatusMerging     DownloadStatus = "merging"
	StatusQueued      DownloadStatus = "queued"
	StatusExtracting  DownloadStatus = "extracting"
)

// Segment represents a download chunk/segment
type Segment struct {
	Index           int    `json:"index"`
	StartByte       int64  `json:"startByte"`
	EndByte         int64  `json:"endByte"`
	DownloadedBytes int64  `json:"downloadedBytes"`
	TempFilePath    string `json:"tempFilePath"`
	Completed       bool   `json:"completed"`
}

// DownloadItem represents a single download entry
type DownloadItem struct {
	ID             string         `json:"id"`
	URL            string         `json:"url"`
	FileName       string         `json:"fileName"`
	SavePath       string         `json:"savePath"`
	TotalSize      int64          `json:"totalSize"`
	DownloadedSize int64          `json:"downloadedSize"`
	Cookies        string         `json:"cookies"`
	Referrer       string         `json:"referrer"`
	Status         DownloadStatus `json:"status"`
	Segments       []Segment      `json:"segments"`
	ThreadCount    int            `json:"threadCount"`
	Speed          int64          `json:"speed"` // bytes per second
	Progress       float64        `json:"progress"`
	Resumable      bool           `json:"resumable"`
	Error          string         `json:"error,omitempty"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
	CompletedAt    *time.Time     `json:"completedAt,omitempty"`

	// Runtime fields (not persisted)
	mu           sync.Mutex     `json:"-"`
	cancelFunc   func()         `json:"-"`
	speedTracker *SpeedTracker  `json:"-"`
	SpeedLimiter ReaderLimiter  `json:"-"`
}

type ReaderLimiter interface {
	Reader(io.Reader) io.Reader
	SetLimit(int64)
}

// Lock locks the download item mutex
func (d *DownloadItem) Lock() {
	d.mu.Lock()
}

// Unlock unlocks the download item mutex
func (d *DownloadItem) Unlock() {
	d.mu.Unlock()
}

// SetCancelFunc sets the cancel function for this download
func (d *DownloadItem) SetCancelFunc(fn func()) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.cancelFunc = fn
}

// Cancel calls the cancel function if set
func (d *DownloadItem) Cancel() {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.cancelFunc != nil {
		d.cancelFunc()
		d.cancelFunc = nil
	}
}

// SpeedTracker tracks download speed
type SpeedTracker struct {
	mu          sync.Mutex
	lastBytes   int64
	lastTime    time.Time
	currentSpeed int64
}

// NewSpeedTracker creates a new speed tracker
func NewSpeedTracker() *SpeedTracker {
	return &SpeedTracker{
		lastTime: time.Now(),
	}
}

// Update updates the speed calculation
func (s *SpeedTracker) Update(totalBytes int64) int64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(s.lastTime).Seconds()
	if elapsed >= 0.5 {
		byteDiff := totalBytes - s.lastBytes
		s.currentSpeed = int64(float64(byteDiff) / elapsed)
		s.lastBytes = totalBytes
		s.lastTime = now
	}
	return s.currentSpeed
}

// GetSpeed returns the current speed
func (s *SpeedTracker) GetSpeed() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.currentSpeed
}

// Settings holds application settings
type Settings struct {
	MaxConcurrentDownloads int    `json:"maxConcurrentDownloads"`
	DefaultThreadCount     int    `json:"defaultThreadCount"`
	DefaultSaveDir         string `json:"defaultSaveDir"`
	SpeedLimitEnabled      bool   `json:"speedLimitEnabled"`
	SpeedLimitBytesPerSec  int64  `json:"speedLimitBytesPerSec"`
	AutoStartDownload      bool   `json:"autoStartDownload"`
	SmartCategorization    bool   `json:"smartCategorization"`
	AutoExtract            bool   `json:"autoExtract"`
	BandwidthMode          string `json:"bandwidthMode"`
	PrioritySecondaryLimit int64  `json:"prioritySecondaryLimit"`
}

// DefaultSettings returns default application settings
func DefaultSettings() Settings {
	return Settings{
		MaxConcurrentDownloads: 3,
		DefaultThreadCount:     8,
		DefaultSaveDir:         "",
		SpeedLimitEnabled:      false,
		SpeedLimitBytesPerSec:  0,
		AutoStartDownload:      true,
		SmartCategorization:    false,
		AutoExtract:            false,
		BandwidthMode:          "flat",
		PrioritySecondaryLimit: 10 * 1024 * 1024,
	}
}

// DownloadProgress is emitted to frontend for real-time updates
type DownloadProgress struct {
	ID             string         `json:"id"`
	DownloadedSize int64          `json:"downloadedSize"`
	TotalSize      int64          `json:"totalSize"`
	Speed          int64          `json:"speed"`
	Progress       float64        `json:"progress"`
	Status         DownloadStatus `json:"status"`
	Error          string         `json:"error,omitempty"`
}
