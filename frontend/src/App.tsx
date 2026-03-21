import React, { useState, useEffect, useCallback } from 'react';
import Toolbar from './components/Toolbar';
import DownloadList from './components/DownloadList';
import AddDownloadModal from './components/AddDownloadModal';
import SettingsPanel from './components/SettingsPanel';
import { DownloadItem, DownloadProgress, Settings } from './types';

const defaultSettings: Settings = {
  maxConcurrentDownloads: 3,
  defaultThreadCount: 8,
  defaultSaveDir: '',
  speedLimitEnabled: false,
  speedLimitBytesPerSec: 0,
  autoStartDownload: true,
  smartCategorization: false,
  autoExtract: false,
  bandwidthMode: 'flat',
  prioritySecondaryLimit: 10485760
};

const App: React.FC = () => {
  const [downloads, setDownloads] = useState<DownloadItem[]>([]);
  const [settings, setSettings] = useState<Settings>(defaultSettings);
  const [isAddModalOpen, setIsAddModalOpen] = useState(false);
  const [isSettingsOpen, setIsSettingsOpen] = useState(false);

  // Initialize data
  useEffect(() => {
    const init = async () => {
      try {
        if (!window.go || !window.go.main || !window.go.main.App) {
          console.warn('Wails backend not available');
          return;
        }

        const [initialDownloads, initialSettings] = await Promise.all([
          window.go.main.App.GetDownloads(),
          window.go.main.App.GetSettings(),
        ]);

        if (initialDownloads) setDownloads(initialDownloads);
        if (initialSettings) setSettings(initialSettings);
      } catch (err) {
        console.error('Failed to initialize:', err);
      }
    };

    init();
  }, []);

  // Set up event listeners for backend updates
  useEffect(() => {
    if (!window.runtime || !window.runtime.EventsOn) return;

    const unsubs: (() => void)[] = [];

    unsubs.push(window.runtime.EventsOn('download:added', (item: DownloadItem) => {
      setDownloads(prev => [item, ...prev]);
    }));

    unsubs.push(window.runtime.EventsOn('download:updated', (item: DownloadItem) => {
      if (!item) return;
      setDownloads(prev => prev.map(d => d.id === item.id ? item : d));
    }));

    unsubs.push(window.runtime.EventsOn('download:removed', (data: { id: string }) => {
      setDownloads(prev => prev.filter(d => d.id !== data.id));
    }));

    unsubs.push(window.runtime.EventsOn('download:progress', (progress: DownloadProgress) => {
      setDownloads(prev => prev.map(d => {
        if (d.id === progress.id) {
          return {
            ...d,
            downloadedSize: progress.downloadedSize,
            speed: progress.speed,
            progress: progress.progress,
            status: progress.status !== 'downloading' ? progress.status : d.status,
          };
        }
        return d;
      }));
    }));

    return () => {
      unsubs.forEach(unsub => unsub());
    };
  }, []);

  // Handlers
  const handleAddDownload = async (url: string, savePath: string, threadCount: number) => {
    if (!window.go) return;
    await window.go.main.App.AddDownload(url, savePath, threadCount, "", "");
  };

  const handlePause = async (id: string) => {
    if (!window.go) return;
    await window.go.main.App.PauseDownload(id);
  };

  const handleResume = async (id: string) => {
    if (!window.go) return;
    await window.go.main.App.ResumeDownload(id);
  };

  const handleCancel = async (id: string) => {
    if (!window.go) return;
    await window.go.main.App.CancelDownload(id);
  };

  const handleRemove = async (id: string) => {
    if (!window.go) return;
    await window.go.main.App.RemoveDownload(id);
  };

  const handleOpenFolder = async (path: string) => {
    if (!window.go) return;
    try {
      await window.go.main.App.OpenFileFolder(path);
    } catch (err) {
      console.error('Failed to open folder:', err);
    }
  };

  const handlePauseAll = async () => {
    if (!window.go) return;
    await window.go.main.App.PauseAll();
  };

  const handleResumeAll = async () => {
    if (!window.go) return;
    await window.go.main.App.ResumeAll();
  };

  const handleSaveSettings = async (newSettings: Settings) => {
    if (!window.go) return;
    await window.go.main.App.UpdateSettings(newSettings);
    setSettings(newSettings);
  };

  // Compute stats
  const activeCount = downloads.filter(d => d.status === 'downloading').length;
  const totalSpeed = downloads.reduce((acc, d) => d.status === 'downloading' ? acc + d.speed : acc, 0);

  return (
    <div className="app">
      <Toolbar
        activeCount={activeCount}
        totalSpeed={totalSpeed}
        onAddClick={() => setIsAddModalOpen(true)}
        onPauseAll={handlePauseAll}
        onResumeAll={handleResumeAll}
        onSettingsClick={() => setIsSettingsOpen(true)}
      />

      <DownloadList
        downloads={downloads}
        onPause={handlePause}
        onResume={handleResume}
        onCancel={handleCancel}
        onRemove={handleRemove}
        onOpenFolder={handleOpenFolder}
      />

      <AddDownloadModal
        isOpen={isAddModalOpen}
        onClose={() => setIsAddModalOpen(false)}
        onSubmit={handleAddDownload}
        defaultSaveDir={settings.defaultSaveDir}
        defaultThreadCount={settings.defaultThreadCount}
      />

      <SettingsPanel
        isOpen={isSettingsOpen}
        onClose={() => setIsSettingsOpen(false)}
        settings={settings}
        onSave={handleSaveSettings}
      />
    </div>
  );
};

export default App;
