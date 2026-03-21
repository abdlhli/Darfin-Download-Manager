import React from 'react';
import { Plus, Pause, Play, Settings, Download, Rocket } from 'lucide-react';

interface Props {
  activeCount: number;
  totalSpeed: number;
  onAddClick: () => void;
  onPauseAll: () => void;
  onResumeAll: () => void;
  onSettingsClick: () => void;
}

const Toolbar: React.FC<Props> = ({
  activeCount,
  totalSpeed,
  onAddClick,
  onPauseAll,
  onResumeAll,
  onSettingsClick,
}) => {
  const formatSpeed = (bytes: number) => {
    if (bytes <= 0) return '0 B/s';
    const k = 1024;
    const sizes = ['B/s', 'KB/s', 'MB/s', 'GB/s'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
  };

  return (
    <div className="toolbar">
      {/* <div className="toolbar__brand">
        <img src="/favicon.png" alt="DARFIN Logo" style={{ width: '28px', height: '28px', marginRight: '8px' }} />
        <span className="toolbar__title" style={{ color: 'white' }}>DARFIN</span>
      </div> */}

      <div className="toolbar__actions">
        <button className="btn btn--primary" onClick={onAddClick} id="btn-add-download" style={{ display: 'flex', alignItems: 'center', gap: '4px' }}>
          <Plus size={16} /> Add URL
        </button>
        <button className="btn" onClick={onPauseAll} title="Pause All" id="btn-pause-all" style={{ display: 'flex', alignItems: 'center', gap: '4px' }}>
          <Pause size={16} /> Pause All
        </button>
        <button className="btn btn--success" onClick={onResumeAll} title="Resume All" id="btn-resume-all" style={{ display: 'flex', alignItems: 'center', gap: '4px' }}>
          <Play size={16} /> Resume All
        </button>
        <button className="btn" onClick={onSettingsClick} title="Settings" id="btn-settings" style={{ display: 'flex', alignItems: 'center', gap: '4px' }}>
          <Settings size={16} /> Settings
        </button>
      </div>

      <div className="toolbar__stats">
        <div className="toolbar__stat">
          <Download size={16} className="toolbar__stat-icon" />
          <span className="toolbar__stat-value">{activeCount}</span>
          <span>active</span>
        </div>
        <div className="toolbar__stat">
          <Rocket size={16} className="toolbar__stat-icon" />
          <span className="toolbar__stat-value">{formatSpeed(totalSpeed)}</span>
        </div>
      </div>
    </div>
  );
};

export default Toolbar;
