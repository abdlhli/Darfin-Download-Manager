import React from 'react';
import { DownloadItem as DownloadItemType } from '../types';
import { formatBytes, formatSpeed, formatTime, estimateTimeRemaining } from '../utils';
import { 
  File, FileArchive, FileImage, FileVideo, FileAudio, FileText, 
  Disc, Settings, Pause, Play, Trash2, X, FolderOpen, RotateCcw, 
  Clock, AlertCircle, CheckCircle2, Download, ArchiveRestore 
} from 'lucide-react';

interface Props {
  item: DownloadItemType;
  onPause: (id: string) => void;
  onResume: (id: string) => void;
  onCancel: (id: string) => void;
  onRemove: (id: string) => void;
  onOpenFolder?: (path: string) => void;
}

const getFileIcon = (fileName: string) => {
  const ext = fileName.split('.').pop()?.toLowerCase() || '';
  if (['zip', 'rar', '7z', 'tar', 'gz'].includes(ext)) return <FileArchive size={24} className="text-gray-500" />;
  if (['jpg', 'jpeg', 'png', 'gif', 'svg', 'webp', 'bmp'].includes(ext)) return <FileImage size={24} className="text-blue-500" />;
  if (['mp4', 'mkv', 'avi', 'mov', 'wmv', 'flv', 'webm'].includes(ext)) return <FileVideo size={24} className="text-purple-500" />;
  if (['mp3', 'wav', 'flac', 'aac', 'ogg', 'wma'].includes(ext)) return <FileAudio size={24} className="text-yellow-500" />;
  if (['pdf', 'doc', 'docx', 'xls', 'xlsx', 'ppt', 'pptx', 'txt', 'csv', 'json', 'xml'].includes(ext)) return <FileText size={24} className="text-red-500" />;
  if (['exe', 'msi', 'dmg', 'deb', 'rpm', 'apk'].includes(ext)) return <Settings size={24} className="text-gray-600" />;
  if (['iso', 'img'].includes(ext)) return <Disc size={24} className="text-gray-400" />;
  return <File size={24} className="text-gray-400" />;
};

const DownloadItem: React.FC<Props> = ({ item, onPause, onResume, onCancel, onRemove, onOpenFolder }) => {
  const progress = Math.min(item.progress, 100);
  const eta = estimateTimeRemaining(item.totalSize, item.downloadedSize, item.speed);

  const statusClass = `download-item--${item.status}`;
  const progressBarClass = `download-item__progress-bar download-item__progress-bar--${item.status}`;

  const renderStatusIcon = () => {
    switch (item.status) {
      case 'downloading': return <Download size={14} className="download-item__status-icon" />;
      case 'paused': return <Pause size={14} className="download-item__status-icon" />;
      case 'completed': return <CheckCircle2 size={14} className="download-item__status-icon" />;
      case 'error': return <AlertCircle size={14} className="download-item__status-icon" />;
      case 'queued': return <Clock size={14} className="download-item__status-icon" />;
      case 'merging': return <RotateCcw size={14} className="download-item__status-icon" />;
      case 'extracting': return <ArchiveRestore size={14} className="download-item__status-icon animate-spin" />;
      default: return null;
    }
  };

  return (
    <div className={`download-item ${statusClass}`}>
      <div className="download-item__header">
        <div className="download-item__info">
          <div className="download-item__icon">
            {getFileIcon(item.fileName)}
          </div>
          <div className="download-item__details">
            <div className="download-item__name" title={item.fileName}>
              {item.fileName}
            </div>
            <div className="download-item__meta">
              <span className={`download-item__status download-item__status--${item.status}`} style={{ display: 'flex', alignItems: 'center', gap: '4px' }}>
                {renderStatusIcon()}
                {item.status}
              </span>
              {item.totalSize > 0 && (
                <span>{formatBytes(item.downloadedSize)} / {formatBytes(item.totalSize)}</span>
              )}
              {item.threadCount > 0 && (
                <span>🧵 {item.threadCount} threads</span>
              )}
            </div>
          </div>
        </div>

        <div className="download-item__actions">
          <button
            className="btn btn--icon btn--sm"
            onClick={() => onOpenFolder?.(item.savePath)}
            title="Show in folder"
          >
            <FolderOpen size={16} />
          </button>
          
          {(item.status === 'downloading') && (
            <button
              className="btn btn--icon btn--sm"
              onClick={() => onPause(item.id)}
              title="Pause"
            >
              <Pause size={16} />
            </button>
          )}
          {(item.status === 'paused' || item.status === 'error') && (
            <button
              className="btn btn--icon btn--sm btn--success"
              onClick={() => onResume(item.id)}
              title="Resume"
            >
              <Play size={16} />
            </button>
          )}
          {(item.status === 'queued') && (
            <button
              className="btn btn--icon btn--sm"
              onClick={() => onPause(item.id)}
              title="Remove from queue"
            >
              <Pause size={16} />
            </button>
          )}
          {item.status !== 'completed' && (
            <button
              className="btn btn--icon btn--sm btn--danger"
              onClick={() => onCancel(item.id)}
              title="Cancel"
            >
              <X size={16} />
            </button>
          )}
          {item.status === 'completed' && (
            <button
              className="btn btn--icon btn--sm btn--danger"
              onClick={() => onRemove(item.id)}
              title="Remove"
            >
              <Trash2 size={16} />
            </button>
          )}
        </div>
      </div>

      {/* Progress bar */}
      <div className="download-item__progress">
        <div
          className={progressBarClass}
          style={{ width: `${progress}%` }}
        />
      </div>

      <div className="download-item__progress-stats">
        <span>
          {item.status === 'downloading' && item.speed > 0 && (
            <span className="download-item__speed">
              {formatSpeed(item.speed)}
            </span>
          )}
          {item.status === 'downloading' && eta > 0 && (
            <span> — ETA {formatTime(eta)}</span>
          )}
          {item.status === 'error' && item.error && (
            <span style={{ color: 'var(--status-error)' }}>{item.error}</span>
          )}
        </span>
        <span>{progress.toFixed(1)}%</span>
      </div>
    </div>
  );
};

export default DownloadItem;
