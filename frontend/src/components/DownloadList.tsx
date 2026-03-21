import React from 'react';
import { DownloadItem as DownloadItemType } from '../types';
import DownloadItem from './DownloadItem';
import { DownloadCloud } from 'lucide-react';

interface Props {
  downloads: DownloadItemType[];
  onPause: (id: string) => void;
  onResume: (id: string) => void;
  onCancel: (id: string) => void;
  onRemove: (id: string) => void;
  onOpenFolder?: (path: string) => void;
}

const DownloadList: React.FC<Props> = ({ downloads, onPause, onResume, onCancel, onRemove, onOpenFolder }) => {
  if (downloads.length === 0) {
    return (
      <div className="download-list">
        <div className="download-list__empty">
          <div className="download-list__empty-icon">
            <DownloadCloud size={48} className="text-gray-400" />
          </div>
          <div className="download-list__empty-text">No downloads yet</div>
          <div className="download-list__empty-hint">
            Click "Add URL" to start downloading files
          </div>
        </div>
      </div>
    );
  }

  // Sort: downloading first, then queued, paused, error, completed
  const statusOrder: Record<string, number> = {
    downloading: 0,
    merging: 1,
    queued: 2,
    paused: 3,
    error: 4,
    completed: 5,
    pending: 6,
  };

  const sorted = [...downloads].sort((a, b) => {
    const orderA = statusOrder[a.status] ?? 99;
    const orderB = statusOrder[b.status] ?? 99;
    if (orderA !== orderB) return orderA - orderB;
    return new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime();
  });

  return (
    <div className="download-list">
      {sorted.map((item) => (
        <DownloadItem
          key={item.id}
          item={item}
          onPause={onPause}
          onResume={onResume}
          onCancel={onCancel}
          onRemove={onRemove}
          onOpenFolder={onOpenFolder}
        />
      ))}
    </div>
  );
};

export default DownloadList;
