package queue

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"darfin/internal/downloader"
	"darfin/internal/models"
	"darfin/internal/store"

	"github.com/google/uuid"
)

// EventEmitter is a callback for emitting events to the frontend
type EventEmitter func(eventName string, data interface{})

// Manager handles download queue, scheduling, and state management
type Manager struct {
	mu           sync.RWMutex
	downloads    []*models.DownloadItem
	engine       *downloader.Engine
	store        *store.Store
	settings     models.Settings
	emitEvent    EventEmitter
	speedLimiter *downloader.SpeedLimiter
	activeCount  int
	cancelFuncs  map[string]context.CancelFunc
}

// NewManager creates a new queue manager
func NewManager(store *store.Store, emitter EventEmitter) (*Manager, error) {
	settings, err := store.LoadSettings()
	if err != nil {
		settings = models.DefaultSettings()
	}

	// Create temp directory
	tempDir := filepath.Join(store.GetConfigDir(), "temp")
	os.MkdirAll(tempDir, 0755)

	m := &Manager{
		store:       store,
		settings:    settings,
		emitEvent:   emitter,
		cancelFuncs: make(map[string]context.CancelFunc),
	}

	// Create speed limiter
	var limitBytes int64
	if settings.SpeedLimitEnabled {
		limitBytes = settings.SpeedLimitBytesPerSec
	}
	m.speedLimiter = downloader.NewSpeedLimiter(limitBytes)

	// Create download engine
	m.engine = downloader.NewEngine(tempDir, settings.DefaultThreadCount, func(progress models.DownloadProgress) {
		if m.emitEvent != nil {
			m.emitEvent("download:progress", progress)
		}
	})
	m.engine.SetSpeedLimiter(m.speedLimiter)

	// Load existing downloads
	downloads, err := store.LoadDownloads()
	if err == nil {
		for i := range downloads {
			dl := downloads[i]
			// Reset downloading items to paused on startup
			if dl.Status == models.StatusDownloading || dl.Status == models.StatusMerging {
				dl.Status = models.StatusPaused
			}
			m.downloads = append(m.downloads, &dl)
		}
	}

	return m, nil
}

// GetDownloads returns all download items
func (m *Manager) GetDownloads() []models.DownloadItem {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]models.DownloadItem, len(m.downloads))
	for i, d := range m.downloads {
		d.Lock()
		result[i] = *d
		d.Unlock()
	}
	return result
}

// getCategoryForFile returns a category folder name based on the file extension
func getCategoryForFile(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".mp4", ".mkv", ".avi", ".mov", ".wmv", ".flv", ".webm":
		return "Video"
	case ".mp3", ".wav", ".flac", ".aac", ".ogg", ".wma":
		return "Music"
	case ".zip", ".rar", ".7z", ".tar", ".gz":
		return "Compressed"
	case ".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx", ".txt", ".csv", ".json", ".xml":
		return "Documents"
	case ".exe", ".msi", ".dmg", ".deb", ".rpm", ".apk":
		return "Programs"
	default:
		return "Other"
	}
}

// AddDownload creates a new download and adds it to the queue
func (m *Manager) AddDownload(url string, savePath string, threadCount int, cookies string, referrer string) (*models.DownloadItem, error) {
	// Probe URL
	fileName, totalSize, resumable, err := m.engine.ProbeURL(url, cookies, referrer)
	if err != nil {
		return nil, fmt.Errorf("failed to probe URL: %w", err)
	}

	if savePath == "" {
		savePath = filepath.Join(m.settings.DefaultSaveDir, fileName)
	}

	// Apply smart categorization if enabled and saving to the default directory
	isDefaultDir := strings.HasPrefix(filepath.Clean(savePath), filepath.Clean(m.settings.DefaultSaveDir))
	if m.settings.SmartCategorization && isDefaultDir {
		category := getCategoryForFile(fileName)
		// Re-build path: DefaultSaveDir / Category / FileName
		saveDir := filepath.Dir(savePath)
		if filepath.Base(saveDir) != category {
			savePath = filepath.Join(m.settings.DefaultSaveDir, category, fileName)
		}
	}

	// Ensure filename from probe is used if savePath is a directory
	info, err := os.Stat(savePath)
	if err == nil && info.IsDir() {
		// Protect against directory traversal
		safeFileName := filepath.Base(filepath.Clean(fileName))
		if safeFileName == "." || safeFileName == "/" || safeFileName == "\\" || safeFileName == "" {
			safeFileName = "download_" + uuid.New().String()[:8]
		}
		savePath = filepath.Join(savePath, safeFileName)
	}

	// Auto-rename if file already exists or is in queue
	originalSavePath := savePath
	ext := filepath.Ext(originalSavePath)
	nameWithoutExt := strings.TrimSuffix(originalSavePath, ext)
	counter := 1

	for {
		_, errStat := os.Stat(savePath)
		isOnDisk := errStat == nil

		isInQueue := false
		m.mu.RLock()
		for _, d := range m.downloads {
			d.Lock()
			inUse := d.SavePath == savePath
			d.Unlock()
			if inUse {
				isInQueue = true
				break
			}
		}
		m.mu.RUnlock()

		if !isOnDisk && !isInQueue {
			break
		}

		savePath = fmt.Sprintf("%s (%d)%s", nameWithoutExt, counter, ext)
		counter++
	}

	if threadCount <= 0 {
		threadCount = m.settings.DefaultThreadCount
	}

	initialStatus := models.StatusQueued
	if !m.settings.AutoStartDownload {
		initialStatus = models.StatusPaused
	}

	item := &models.DownloadItem{
		ID:          uuid.New().String()[:8],
		URL:         url,
		FileName:    filepath.Base(savePath),
		SavePath:    savePath,
		TotalSize:   totalSize,
		Status:      initialStatus,
		ThreadCount: threadCount,
		Resumable:   resumable,
		Cookies:     cookies,
		Referrer:    referrer,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	m.mu.Lock()
	m.downloads = append(m.downloads, item)
	m.mu.Unlock()

	m.saveState()
	m.emitEvent("download:added", item)

	// Auto-start if enabled
	if m.settings.AutoStartDownload {
		go m.tryStartNext()
	}

	return item, nil
}

// PauseDownload pauses an active download
func (m *Manager) PauseDownload(id string) error {
	m.mu.Lock()
	item := m.findByID(id)
	if item == nil {
		m.mu.Unlock()
		return fmt.Errorf("download not found: %s", id)
	}

	if cancel, ok := m.cancelFuncs[id]; ok {
		cancel()
		delete(m.cancelFuncs, id)
	}

	item.Lock()
	item.Status = models.StatusPaused
	item.Speed = 0
	item.UpdatedAt = time.Now()
	item.Unlock()
	m.mu.Unlock()

	m.saveState()
	m.emitEvent("download:updated", m.getItemCopy(id))
	go m.tryStartNext()

	return nil
}

// ResumeDownload resumes a paused download
func (m *Manager) ResumeDownload(id string) error {
	m.mu.Lock()
	item := m.findByID(id)
	if item == nil {
		m.mu.Unlock()
		return fmt.Errorf("download not found: %s", id)
	}

	item.Lock()
	if item.Status != models.StatusPaused && item.Status != models.StatusError {
		item.Unlock()
		m.mu.Unlock()
		return fmt.Errorf("download is not paused or errored")
	}
	item.Status = models.StatusQueued
	item.Error = ""
	item.UpdatedAt = time.Now()
	item.Unlock()
	m.mu.Unlock()

	m.saveState()
	m.emitEvent("download:updated", m.getItemCopy(id))
	go m.tryStartNext()

	return nil
}

// CancelDownload cancels and removes a download
func (m *Manager) CancelDownload(id string) error {
	m.mu.Lock()
	item := m.findByID(id)
	if item == nil {
		m.mu.Unlock()
		return fmt.Errorf("download not found: %s", id)
	}

	if cancel, ok := m.cancelFuncs[id]; ok {
		cancel()
		delete(m.cancelFuncs, id)
	}

	item.Lock()
	// Clean up temp file
	os.Remove(item.SavePath + ".darfin")
	item.Unlock()

	m.mu.Unlock()

	return m.RemoveDownload(id)
}

// RemoveDownload removes a download from the list
func (m *Manager) RemoveDownload(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	idx := -1
	for i, d := range m.downloads {
		if d.ID == id {
			idx = i
			break
		}
	}

	if idx == -1 {
		return fmt.Errorf("download not found: %s", id)
	}

	// Cancel if active
	if cancel, ok := m.cancelFuncs[id]; ok {
		cancel()
		delete(m.cancelFuncs, id)
	}

	m.downloads = append(m.downloads[:idx], m.downloads[idx+1:]...)
	m.saveStateUnsafe()
	m.updateBandwidthPrioritiesUnsafe()
	m.emitEvent("download:removed", map[string]string{"id": id})

	return nil
}

// PauseAll pauses all active downloads
func (m *Manager) PauseAll() {
	m.mu.RLock()
	ids := make([]string, 0)
	for _, d := range m.downloads {
		d.Lock()
		if d.Status == models.StatusDownloading {
			ids = append(ids, d.ID)
		}
		d.Unlock()
	}
	m.mu.RUnlock()

	for _, id := range ids {
		m.PauseDownload(id)
	}
}

// ResumeAll resumes all paused downloads
func (m *Manager) ResumeAll() {
	m.mu.RLock()
	ids := make([]string, 0)
	for _, d := range m.downloads {
		d.Lock()
		if d.Status == models.StatusPaused {
			ids = append(ids, d.ID)
		}
		d.Unlock()
	}
	m.mu.RUnlock()

	for _, id := range ids {
		m.ResumeDownload(id)
	}
}

// tryStartNext starts the next queued download if there's a free slot
func (m *Manager) tryStartNext() {
	m.mu.Lock()

	// Count active downloads
	active := 0
	for _, d := range m.downloads {
		d.Lock()
		if d.Status == models.StatusDownloading {
			active++
		}
		d.Unlock()
	}

	// Find next queued item
	var nextItem *models.DownloadItem
	for _, d := range m.downloads {
		d.Lock()
		if d.Status == models.StatusQueued {
			nextItem = d
			d.Unlock()
			break
		}
		d.Unlock()
	}

	if nextItem == nil || active >= m.settings.MaxConcurrentDownloads {
		m.mu.Unlock()
		return
	}

	// Start downloading
	ctx, cancel := context.WithCancel(context.Background())
	m.cancelFuncs[nextItem.ID] = cancel

	nextItem.Lock()
	nextItem.Status = models.StatusDownloading
	nextItem.UpdatedAt = time.Now()
	nextItem.Unlock()

	m.mu.Unlock()

	m.updateBandwidthPriorities()
	m.saveState()
	if copyItem := m.getItemCopy(nextItem.ID); copyItem != nil {
		m.emitEvent("download:updated", copyItem)
	}

	// Start download in goroutine
	go func(item *models.DownloadItem) {
		err := m.engine.StartDownload(ctx, item)

		item.Lock()
		if err != nil {
			if ctx.Err() != nil {
				// Cancelled/paused — status already set
			} else {
				item.Status = models.StatusError
				item.Error = err.Error()
			}
		} else {
			if m.settings.AutoExtract && strings.ToLower(filepath.Ext(item.FileName)) == ".zip" {
				item.Status = models.StatusExtracting
				item.UpdatedAt = time.Now()
				item.Unlock()

				// Tell UI we are extracting
				m.emitEvent("download:updated", m.getItemCopy(item.ID))

				extractErr := extractZip(item.SavePath)

				item.Lock()
				if extractErr != nil {
					item.Status = models.StatusError
					item.Error = fmt.Sprintf("Extraction failed: %v", extractErr)
				} else {
					item.Status = models.StatusCompleted
					item.Progress = 100
					now := time.Now()
					item.CompletedAt = &now
				}
			} else {
				item.Status = models.StatusCompleted
				item.Progress = 100
				now := time.Now()
				item.CompletedAt = &now
			}
		}
		item.Speed = 0
		item.UpdatedAt = time.Now()
		item.Unlock()

		m.mu.Lock()
		delete(m.cancelFuncs, item.ID)
		m.mu.Unlock()

		m.updateBandwidthPriorities()
		m.saveState()
		if copyItem := m.getItemCopy(item.ID); copyItem != nil {
			m.emitEvent("download:updated", copyItem)
		}

		// Try to start next queued item
		m.tryStartNext()
	}(nextItem)
}

// GetSettings returns current settings
func (m *Manager) GetSettings() models.Settings {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.settings
}

// UpdateSettings updates application settings
func (m *Manager) UpdateSettings(settings models.Settings) error {
	m.mu.Lock()
	m.settings = settings
	m.mu.Unlock()

	// Update speed limiter
	if settings.SpeedLimitEnabled {
		m.speedLimiter.SetLimit(settings.SpeedLimitBytesPerSec)
	} else {
		m.speedLimiter.SetLimit(0)
	}

	m.updateBandwidthPrioritiesUnsafe()

	return m.store.SaveSettings(settings)
}

// findByID finds a download item by ID (must hold mu lock)
func (m *Manager) findByID(id string) *models.DownloadItem {
	for _, d := range m.downloads {
		if d.ID == id {
			return d
		}
	}
	return nil
}

// getItemCopy returns a copy of a download item by ID
func (m *Manager) getItemCopy(id string) *models.DownloadItem {
	m.mu.RLock()
	defer m.mu.RUnlock()
	item := m.findByID(id)
	if item == nil {
		return nil
	}
	item.Lock()
	copy := *item
	item.Unlock()
	return &copy
}

// saveState persists current downloads (thread-safe)
func (m *Manager) saveState() {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.saveStateUnsafe()
}

// saveStateUnsafe persists current downloads (caller must hold lock)
func (m *Manager) saveStateUnsafe() {
	items := make([]models.DownloadItem, len(m.downloads))
	for i, d := range m.downloads {
		d.Lock()
		items[i] = *d
		d.Unlock()
	}
	m.store.SaveDownloads(items)
}

// Shutdown gracefully stops all downloads
func (m *Manager) Shutdown() {
	m.mu.Lock()
	for id, cancel := range m.cancelFuncs {
		cancel()
		delete(m.cancelFuncs, id)
	}

	// Mark all downloading items as paused
	for _, d := range m.downloads {
		d.Lock()
		if d.Status == models.StatusDownloading {
			d.Status = models.StatusPaused
			d.Speed = 0
		}
		d.Unlock()
	}
	m.mu.Unlock()

	m.saveState()
}

func (m *Manager) updateBandwidthPriorities() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.updateBandwidthPrioritiesUnsafe()
}

func (m *Manager) updateBandwidthPrioritiesUnsafe() {
	if m.settings.BandwidthMode != "priority" {
		for _, d := range m.downloads {
			d.Lock()
			if d.SpeedLimiter != nil {
				d.SpeedLimiter.SetLimit(0)
			}
			d.Unlock()
		}
		return
	}

	var activeItems []*models.DownloadItem
	for _, d := range m.downloads {
		d.Lock()
		if d.Status == models.StatusDownloading {
			activeItems = append(activeItems, d)
		}
		d.Unlock()
	}

	if len(activeItems) == 0 {
		return
	}

	first := activeItems[0]
	first.Lock()
	if first.SpeedLimiter != nil {
		first.SpeedLimiter.SetLimit(0)
	}
	first.Unlock()

	secondaryLimit := m.settings.PrioritySecondaryLimit
	if secondaryLimit <= 0 {
		secondaryLimit = 10 * 1024 * 1024
	}

	for i := 1; i < len(activeItems); i++ {
		item := activeItems[i]
		item.Lock()
		if item.SpeedLimiter == nil {
			item.SpeedLimiter = downloader.NewSpeedLimiter(secondaryLimit)
		} else {
			item.SpeedLimiter.SetLimit(secondaryLimit)
		}
		item.Unlock()
	}
}
