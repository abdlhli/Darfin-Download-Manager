import React, { useState, useEffect, useCallback } from 'react';
import { Download, FolderOpen, Loader2 } from 'lucide-react';

interface Props {
  isOpen: boolean;
  onClose: () => void;
  onSubmit: (url: string, savePath: string, threadCount: number) => void;
  defaultSaveDir: string;
  defaultThreadCount: number;
}

const AddDownloadModal: React.FC<Props> = ({ isOpen, onClose, onSubmit, defaultSaveDir, defaultThreadCount }) => {
  const [url, setUrl] = useState('');
  const [savePath, setSavePath] = useState(defaultSaveDir);
  const [threadCount, setThreadCount] = useState(defaultThreadCount);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    setSavePath(defaultSaveDir);
    setThreadCount(defaultThreadCount);
  }, [defaultSaveDir, defaultThreadCount]);

  // Auto-paste from clipboard when modal opens
  useEffect(() => {
    if (isOpen) {
      navigator.clipboard.readText().then((text) => {
        if (text && (text.startsWith('http://') || text.startsWith('https://') || text.startsWith('ftp://'))) {
          setUrl(text);
        }
      }).catch(() => {});
    }
  }, [isOpen]);

  const handleBrowse = useCallback(async () => {
    try {
      const dir = await window.go.main.App.OpenDirectoryDialog();
      if (dir) {
        setSavePath(dir);
      }
    } catch (e) {
      console.error('Failed to open directory dialog:', e);
    }
  }, []);

  const handleSubmit = useCallback(async (e: React.FormEvent) => {
    e.preventDefault();
    if (!url.trim()) return;

    setLoading(true);
    try {
      await onSubmit(url.trim(), savePath, threadCount);
      setUrl('');
      onClose();
    } catch (err) {
      console.error('Failed to add download:', err);
    } finally {
      setLoading(false);
    }
  }, [url, savePath, threadCount, onSubmit, onClose]);

  const handleKeyDown = useCallback((e: React.KeyboardEvent) => {
    if (e.key === 'Escape') {
      onClose();
    }
  }, [onClose]);

  if (!isOpen) return null;

  return (
    <div className="modal-overlay" onClick={onClose} onKeyDown={handleKeyDown}>
      <div className="modal" onClick={(e) => e.stopPropagation()}>
        <div className="modal__header">
          <h2 className="modal__title" style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
            <Download size={20} /> Add Download
          </h2>
          <button className="modal__close" onClick={onClose}>✕</button>
        </div>

        <form onSubmit={handleSubmit}>
          <div className="modal__body">
            <div className="form-group">
              <label className="form-label" htmlFor="download-url">URL</label>
              <input
                id="download-url"
                className="form-input"
                type="url"
                placeholder="https://example.com/file.zip"
                value={url}
                onChange={(e) => setUrl(e.target.value)}
                autoFocus
                required
              />
            </div>

            <div className="form-group">
              <label className="form-label" htmlFor="save-path">Save Location</label>
              <div className="form-input-group">
                <input
                  id="save-path"
                  className="form-input"
                  type="text"
                  placeholder="Download folder path"
                  value={savePath}
                  onChange={(e) => setSavePath(e.target.value)}
                />
                <button type="button" className="btn" onClick={handleBrowse} style={{ display: 'flex', alignItems: 'center', gap: '4px' }}>
                  <FolderOpen size={16} /> Browse
                </button>
              </div>
            </div>

            <div className="form-row">
              <div className="form-group">
                <label className="form-label" htmlFor="thread-count">Threads</label>
                <input
                  id="thread-count"
                  className="form-input settings-number"
                  type="number"
                  min={1}
                  max={32}
                  value={threadCount}
                  onChange={(e) => setThreadCount(parseInt(e.target.value) || 1)}
                />
              </div>
            </div>
          </div>

          <div className="modal__footer">
            <button type="button" className="btn" onClick={onClose}>
              Cancel
            </button>
            <button 
              type="submit" 
              className="btn btn--primary" 
              disabled={loading || !url.trim()}
              style={{ display: 'flex', alignItems: 'center', gap: '4px', justifyContent: 'center' }}
            >
              {loading ? (
                <><Loader2 size={16} className="animate-spin" /> Adding...</>
              ) : (
                <><Download size={16} /> Download</>
              )}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};

export default AddDownloadModal;
