export function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

export function formatSpeed(bytesPerSec: number): string {
  if (bytesPerSec <= 0) return '0 B/s';
  return formatBytes(bytesPerSec) + '/s';
}

export function formatTime(seconds: number): string {
  if (seconds <= 0 || !isFinite(seconds)) return '--:--';
  const h = Math.floor(seconds / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  const s = Math.floor(seconds % 60);
  if (h > 0) return `${h}:${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`;
  return `${m}:${String(s).padStart(2, '0')}`;
}

export function getFileIcon(fileName: string): string {
  const ext = fileName.split('.').pop()?.toLowerCase() || '';
  const icons: Record<string, string> = {
    // Archives
    zip: '📦', rar: '📦', '7z': '📦', tar: '📦', gz: '📦',
    // Images
    jpg: '🖼️', jpeg: '🖼️', png: '🖼️', gif: '🖼️', svg: '🖼️', webp: '🖼️', bmp: '🖼️',
    // Videos
    mp4: '🎬', mkv: '🎬', avi: '🎬', mov: '🎬', wmv: '🎬', flv: '🎬', webm: '🎬',
    // Audio
    mp3: '🎵', wav: '🎵', flac: '🎵', aac: '🎵', ogg: '🎵', wma: '🎵',
    // Documents
    pdf: '📄', doc: '📝', docx: '📝', xls: '📊', xlsx: '📊', ppt: '📎', pptx: '📎',
    txt: '📄', csv: '📊', json: '📄', xml: '📄',
    // Executables
    exe: '⚙️', msi: '⚙️', dmg: '⚙️', deb: '⚙️', rpm: '⚙️', apk: '⚙️',
    // Disk images
    iso: '💿', img: '💿',
  };
  return icons[ext] || '📁';
}

export function getStatusColor(status: string): string {
  const colors: Record<string, string> = {
    downloading: 'var(--status-downloading)',
    paused: 'var(--status-paused)',
    completed: 'var(--status-completed)',
    error: 'var(--status-error)',
    queued: 'var(--status-queued)',
    merging: 'var(--status-merging)',
    pending: 'var(--text-tertiary)',
  };
  return colors[status] || 'var(--text-tertiary)';
}

export function estimateTimeRemaining(totalSize: number, downloadedSize: number, speed: number): number {
  if (speed <= 0 || totalSize <= 0) return 0;
  return (totalSize - downloadedSize) / speed;
}
