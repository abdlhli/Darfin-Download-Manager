import React, { useState, useEffect, useCallback } from 'react';
import { Settings } from '../types';
import { Settings as SettingsIcon, FolderOpen, Save } from 'lucide-react';

interface Props {
  isOpen: boolean;
  onClose: () => void;
  settings: Settings;
  onSave: (settings: Settings) => void;
}

const SettingsPanel: React.FC<Props> = ({ isOpen, onClose, settings, onSave }) => {
  const [local, setLocal] = useState<Settings>(settings);

  useEffect(() => {
    setLocal(settings);
  }, [settings]);

  const handleSave = useCallback(() => {
    onSave(local);
    onClose();
  }, [local, onSave, onClose]);

  const handleBrowseDir = useCallback(async () => {
    try {
      const dir = await window.go.main.App.OpenDirectoryDialog();
      if (dir) {
        setLocal(prev => ({ ...prev, defaultSaveDir: dir }));
      }
    } catch (e) {
      console.error('Failed to open directory dialog:', e);
    }
  }, []);

  const formatSpeedLimit = (bytesPerSec: number): string => {
    if (bytesPerSec <= 0) return 'Unlimited';
    const kb = bytesPerSec / 1024;
    if (kb >= 1024) return `${(kb / 1024).toFixed(1)} MB/s`;
    return `${kb.toFixed(0)} KB/s`;
  };

  if (!isOpen) return null;

  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="modal" onClick={(e) => e.stopPropagation()} style={{ width: '560px' }}>
        <div className="modal__header">
          <h2 className="modal__title" style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
            <SettingsIcon size={20} /> Settings
          </h2>
          <button className="modal__close" onClick={onClose}>✕</button>
        </div>

        <div className="modal__body">
          {/* Download Settings */}
          <div className="settings__section">
            <div className="settings__section-title">Download</div>

            <div className="settings__row">
              <span className="settings__label">Max Concurrent Downloads</span>
              <div className="settings__value">
                <input
                  className="form-input settings-number"
                  type="number"
                  min={1}
                  max={10}
                  value={local.maxConcurrentDownloads}
                  onChange={(e) => setLocal(prev => ({
                    ...prev,
                    maxConcurrentDownloads: parseInt(e.target.value) || 1,
                  }))}
                />
              </div>
            </div>

            <div className="settings__row">
              <span className="settings__label">Default Thread Count</span>
              <div className="settings__value">
                <input
                  className="form-input settings-number"
                  type="number"
                  min={1}
                  max={32}
                  value={local.defaultThreadCount}
                  onChange={(e) => setLocal(prev => ({
                    ...prev,
                    defaultThreadCount: parseInt(e.target.value) || 1,
                  }))}
                />
              </div>
            </div>

            <div className="settings__row">
              <span className="settings__label">Auto Start Download</span>
              <div className="settings__value">
                <div
                  className={`toggle ${local.autoStartDownload ? 'toggle--active' : ''}`}
                  onClick={() => setLocal(prev => ({
                    ...prev,
                    autoStartDownload: !prev.autoStartDownload,
                  }))}
                >
                  <div className="toggle__knob" />
                </div>
              </div>
            </div>

            <div className="settings__row">
              <span className="settings__label">Auto-Extract ZIP archives</span>
              <div className="settings__value">
                <div
                  className={`toggle ${local.autoExtract ? 'toggle--active' : ''}`}
                  onClick={() => setLocal(prev => ({
                    ...prev,
                    autoExtract: !prev.autoExtract,
                  }))}
                >
                  <div className="toggle__knob" />
                </div>
              </div>
            </div>
          </div>

          {/* Save Location */}
          <div className="settings__section">
            <div className="settings__section-title">Save Location</div>
            <div className="form-group">
              <div className="form-input-group">
                <input
                  className="form-input"
                  type="text"
                  placeholder="Default download folder"
                  value={local.defaultSaveDir}
                  onChange={(e) => setLocal(prev => ({ ...prev, defaultSaveDir: e.target.value }))}
                />
                <button type="button" className="btn" onClick={handleBrowseDir} style={{ display: 'flex', alignItems: 'center', gap: '4px' }}>
                  <FolderOpen size={16} /> Browse
                </button>
              </div>
            </div>

            <div className="settings__row" style={{ marginTop: '16px' }}>
              <span className="settings__label">Smart Categorization (Video, Music, etc.)</span>
              <div className="settings__value">
                <div
                  className={`toggle ${local.smartCategorization ? 'toggle--active' : ''}`}
                  onClick={() => setLocal(prev => ({
                    ...prev,
                    smartCategorization: !prev.smartCategorization,
                  }))}
                >
                  <div className="toggle__knob" />
                </div>
              </div>
            </div>
          </div>

          {/* Speed Limit */}
          <div className="settings__section">
            <div className="settings__section-title">Speed Limit</div>

            <div className="settings__row">
              <span className="settings__label">
                Enable Speed Limit
                {local.speedLimitEnabled && (
                  <span style={{ color: 'var(--text-tertiary)', marginLeft: '8px' }}>
                    ({formatSpeedLimit(local.speedLimitBytesPerSec)})
                  </span>
                )}
              </span>
              <div className="settings__value">
                <div
                  className={`toggle ${local.speedLimitEnabled ? 'toggle--active' : ''}`}
                  onClick={() => setLocal(prev => ({
                    ...prev,
                    speedLimitEnabled: !prev.speedLimitEnabled,
                  }))}
                >
                  <div className="toggle__knob" />
                </div>
              </div>
            </div>

            {local.speedLimitEnabled && (
              <div className="settings__row">
                <span className="settings__label">Max Speed (KB/s)</span>
                <div className="settings__value">
                  <input
                    className="form-input settings-number"
                    type="number"
                    min={0}
                    step={100}
                    value={Math.round(local.speedLimitBytesPerSec / 1024)}
                    onChange={(e) => setLocal(prev => ({
                      ...prev,
                      speedLimitBytesPerSec: (parseInt(e.target.value) || 0) * 1024,
                    }))}
                    style={{ width: '120px' }}
                  />
                </div>
              </div>
            )}
          </div>

          {/* Bandwidth Priority */}
          <div className="settings__section">
            <div className="settings__section-title">Bandwidth Allocation (QoS)</div>

            <div className="settings__row">
              <span className="settings__label">Allocation Mode</span>
              <div className="settings__value">
                <select
                  className="form-input"
                  style={{ padding: '6px' }}
                  value={local.bandwidthMode || 'flat'}
                  onChange={(e) => setLocal(prev => ({
                    ...prev,
                    bandwidthMode: e.target.value,
                  }))}
                >
                  <option value="flat">Flat (Equal Sharing)</option>
                  <option value="priority">Priority Active</option>
                </select>
              </div>
            </div>

            {local.bandwidthMode === 'priority' && (
              <div className="settings__row">
                <span className="settings__label">Secondary Downloads Limit (MB/s)</span>
                <div className="settings__value">
                  <input
                    className="form-input settings-number"
                    type="number"
                    min={1}
                    max={100}
                    value={Math.round((local.prioritySecondaryLimit || 10485760) / (1024 * 1024))}
                    onChange={(e) => setLocal(prev => ({
                      ...prev,
                      prioritySecondaryLimit: (parseInt(e.target.value) || 10) * 1024 * 1024,
                    }))}
                    style={{ width: '120px' }}
                  />
                </div>
              </div>
            )}
          </div>
        </div>

        <div className="modal__footer">
          <button className="btn" onClick={onClose}>Cancel</button>
          <button className="btn btn--primary" onClick={handleSave} style={{ display: 'flex', alignItems: 'center', gap: '4px' }}>
            <Save size={16} /> Save Settings
          </button>
        </div>
      </div>
    </div>
  );
};

export default SettingsPanel;
