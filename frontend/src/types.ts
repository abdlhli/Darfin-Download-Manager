/// <reference types="vite/client" />

// Wails runtime types
declare global {
  interface Window {
    go: {
      main: {
        App: {
          AddDownload(url: string, savePath: string, threadCount: number): Promise<DownloadItem>;
          PauseDownload(id: string): Promise<void>;
          ResumeDownload(id: string): Promise<void>;
          CancelDownload(id: string): Promise<void>;
          RemoveDownload(id: string): Promise<void>;
          GetDownloads(): Promise<DownloadItem[]>;
          PauseAll(): Promise<void>;
          ResumeAll(): Promise<void>;
          GetSettings(): Promise<Settings>;
          UpdateSettings(settings: Settings): Promise<void>;
          OpenDirectoryDialog(): Promise<string>;
          OpenFileDialog(defaultFilename: string): Promise<string>;
          OpenFileFolder(filePath: string): Promise<void>;
          GetDefaultSaveDir(): Promise<string>;
          HandleBrowserExtension(url: string, filename: string): Promise<DownloadItem>;
        };
      };
    };
    runtime: {
      EventsOn(eventName: string, callback: (...args: any[]) => void): () => void;
      EventsOff(eventName: string): void;
    };
  }
}

export interface DownloadItem {
  id: string;
  url: string;
  fileName: string;
  savePath: string;
  totalSize: number;
  downloadedSize: number;
  status: DownloadStatus;
  segments: Segment[];
  threadCount: number;
  speed: number;
  progress: number;
  resumable: boolean;
  error?: string;
  createdAt: string;
  updatedAt: string;
  completedAt?: string;
}

export interface Segment {
  index: number;
  startByte: number;
  endByte: number;
  downloadedBytes: number;
  tempFilePath: string;
  completed: boolean;
}

export type DownloadStatus = 'pending' | 'downloading' | 'paused' | 'completed' | 'error' | 'merging' | 'queued' | 'extracting';

export interface DownloadProgress {
  id: string;
  downloadedSize: number;
  totalSize: number;
  speed: number;
  progress: number;
  status: DownloadStatus;
  error?: string;
}

export interface Settings {
  maxConcurrentDownloads: number;
  defaultThreadCount: number;
  defaultSaveDir: string;
  speedLimitEnabled: boolean;
  speedLimitBytesPerSec: number;
  autoStartDownload: boolean;
  smartCategorization: boolean;
  autoExtract: boolean;
}
