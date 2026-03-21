package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	_ "embed"
	"os/exec"
	"path/filepath"

	"darfin/internal/models"
	"darfin/internal/queue"
	"darfin/internal/store"

	"github.com/energye/systray"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed build/windows/icon.ico
var iconData []byte

// App struct holds application state
type App struct {
	ctx     context.Context
	store   *store.Store
	manager *queue.Manager
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Initialize store
	s, err := store.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize store: %v\n", err)
		return
	}
	a.store = s

	// Initialize queue manager with event emitter
	m, err := queue.NewManager(s, func(eventName string, data interface{}) {
		runtime.EventsEmit(ctx, eventName, data)
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize queue manager: %v\n", err)
		return
	}
	a.manager = m

	// Start local server for browser extension
	go a.startExtensionServer()

	// Initialize System Tray
	go systray.Run(func() {
		systray.SetIcon(iconData)
		systray.SetTitle("DARFIN")
		systray.SetTooltip("DARFIN Download Manager")
		
		mShow := systray.AddMenuItem("Show DARFIN", "Show Download Manager")
		mQuit := systray.AddMenuItem("Quit", "Quit Application")

		mShow.Click(func() {
			runtime.WindowShow(ctx)
		})
		mQuit.Click(func() {
			systray.Quit()
			runtime.Quit(ctx)
		})
	}, func() {})
}

// shutdown is called when the app exits
func (a *App) shutdown(ctx context.Context) {
	if a.manager != nil {
		a.manager.Shutdown()
	}
}

// AddDownload adds a new download to the queue
func (a *App) AddDownload(url string, savePath string, threadCount int) (*models.DownloadItem, error) {
	if a.manager == nil {
		return nil, fmt.Errorf("manager not initialized")
	}

	// Ensure the application window is brought to the front and unminimized
	runtime.WindowUnminimise(a.ctx)
	runtime.WindowShow(a.ctx)

	return a.manager.AddDownload(url, savePath, threadCount)
}

// PauseDownload pauses an active download
func (a *App) PauseDownload(id string) error {
	if a.manager == nil {
		return fmt.Errorf("manager not initialized")
	}
	return a.manager.PauseDownload(id)
}

// ResumeDownload resumes a paused download
func (a *App) ResumeDownload(id string) error {
	if a.manager == nil {
		return fmt.Errorf("manager not initialized")
	}
	return a.manager.ResumeDownload(id)
}

// CancelDownload cancels a download
func (a *App) CancelDownload(id string) error {
	if a.manager == nil {
		return fmt.Errorf("manager not initialized")
	}
	return a.manager.CancelDownload(id)
}

// RemoveDownload removes a completed/failed download from the list
func (a *App) RemoveDownload(id string) error {
	if a.manager == nil {
		return fmt.Errorf("manager not initialized")
	}
	return a.manager.RemoveDownload(id)
}

// GetDownloads returns all downloads
func (a *App) GetDownloads() []models.DownloadItem {
	if a.manager == nil {
		return []models.DownloadItem{}
	}
	return a.manager.GetDownloads()
}

// PauseAll pauses all active downloads
func (a *App) PauseAll() {
	if a.manager != nil {
		a.manager.PauseAll()
	}
}

// ResumeAll resumes all paused downloads
func (a *App) ResumeAll() {
	if a.manager != nil {
		a.manager.ResumeAll()
	}
}

// GetSettings returns application settings
func (a *App) GetSettings() models.Settings {
	if a.manager == nil {
		return models.DefaultSettings()
	}
	return a.manager.GetSettings()
}

// UpdateSettings updates application settings
func (a *App) UpdateSettings(settings models.Settings) error {
	if a.manager == nil {
		return fmt.Errorf("manager not initialized")
	}
	return a.manager.UpdateSettings(settings)
}

// OpenDirectoryDialog opens a native directory picker dialog
func (a *App) OpenDirectoryDialog() string {
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Download Folder",
	})
	if err != nil {
		return ""
	}
	return dir
}

// OpenFileDialog opens a native file save dialog
func (a *App) OpenFileDialog(defaultFilename string) string {
	filePath, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Save File As",
		DefaultFilename: defaultFilename,
	})
	if err != nil {
		return ""
	}
	return filePath
}

// OpenFileFolder opens the system file explorer and selects the file
func (a *App) OpenFileFolder(filePath string) error {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", filePath)
	}
	// Use explorer /select to highlight the file in its containing folder
	cmd := exec.Command("explorer", "/select,", filePath)
	return cmd.Start()
}

// GetDefaultSaveDir returns the default download directory
func (a *App) GetDefaultSaveDir() string {
	settings := a.GetSettings()
	if settings.DefaultSaveDir != "" {
		return settings.DefaultSaveDir
	}
	// Use user's Downloads folder as default
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Downloads")
}

// HandleBrowserExtension handles download requests from browser extension
func (a *App) HandleBrowserExtension(url string, filename string) (*models.DownloadItem, error) {
	savePath := filepath.Join(a.GetDefaultSaveDir(), filename)
	return a.AddDownload(url, savePath, 0)
}

type ExtensionRequest struct {
	URL      string `json:"url"`
	Filename string `json:"filename"`
}

// startExtensionServer starts a local HTTP server for the browser extension to send downloads to
func (a *App) startExtensionServer() {
	http.HandleFunc("/add", func(w http.ResponseWriter, r *http.Request) {
		// Allow CORS
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req ExtensionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		if req.URL != "" {
			go a.HandleBrowserExtension(req.URL, filepath.Base(req.Filename))
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	if err := http.ListenAndServe("127.0.0.1:3547", nil); err != nil {
		fmt.Println("Local extension server error:", err)
	}
}
