document.addEventListener('DOMContentLoaded', () => {
  const toggleCapture = document.getElementById('toggle-capture');

  try {
    const storageAPI = (typeof browser !== 'undefined' && browser.storage) ? browser.storage.local : (chrome && chrome.storage ? chrome.storage.local : null);
    
    // Load current state
    if (storageAPI) {
      storageAPI.get(['isEnabled'], (result) => {
        if (result && result.isEnabled !== undefined) {
          toggleCapture.checked = result.isEnabled;
        }
      });

      // Save state on change
      toggleCapture.addEventListener('change', (e) => {
        storageAPI.set({ isEnabled: e.target.checked });
      });
    } else {
      console.log("Storage API unavailable. UI changes won't be saved.");
    }
  } catch (e) {
    console.log("Error accessing storage:", e);
  }

  // Load captured media
  const mediaList = document.getElementById('media-list');
  const mediaCount = document.getElementById('media-count');
  
  try {
    const rAPI = (typeof browser !== 'undefined' && browser.runtime) ? browser.runtime : chrome.runtime;
    const tAPI = (typeof browser !== 'undefined' && browser.tabs) ? browser.tabs : chrome.tabs;
    
    rAPI.sendMessage({ action: "get-captured-media" }, (response) => {
      if (rAPI.lastError) return;
      
      const media = response?.media || [];
      mediaCount.textContent = media.length;
      
      if (media.length > 0) {
        mediaList.innerHTML = '';
        
        media.forEach(item => {
          const div = document.createElement('div');
          div.style.cssText = 'padding: 8px; background: #f8f9fa; border-radius: 4px; margin-bottom: 6px; display: flex; flex-direction: column; gap: 6px; border: 1px solid #eee;';
          
          const title = document.createElement('div');
          title.textContent = item.title || 'Unknown Media';
          title.style.cssText = 'font-weight: 500; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; color: #333;';
          
          const meta = document.createElement('div');
          meta.style.cssText = 'display: flex; justify-content: space-between; align-items: center;';
          
          const badge = document.createElement('span');
          badge.textContent = item.ext.toUpperCase();
          badge.style.cssText = 'background: #6c5ce7; color: white; padding: 2px 4px; border-radius: 3px; font-size: 9px; font-weight: bold; letter-spacing: 0.5px;';
          
          const btn = document.createElement('button');
          btn.textContent = 'Download';
          btn.style.cssText = 'background: #00b894; color: white; border: none; padding: 4px 10px; border-radius: 4px; cursor: pointer; font-size: 11px; font-weight: 500; transition: background 0.2s;';
          
          btn.onmouseover = () => btn.style.background = '#00a884';
          btn.onmouseout = () => btn.style.background = '#00b894';
          
          btn.onclick = () => {
            btn.textContent = 'Sent ✓';
            btn.style.background = '#b2bec3';
            btn.style.cursor = 'default';
            btn.disabled = true;
            
            tAPI.query({active: true, currentWindow: true}, function(tabs) {
              const referrer = tabs[0] ? tabs[0].url : "";
              rAPI.sendMessage({
                action: "darfin-download-media",
                url: item.url,
                filename: `video.${item.ext}`,
                referrer: referrer
              });
            });
          };
          
          meta.appendChild(badge);
          meta.appendChild(btn);
          div.appendChild(title);
          div.appendChild(meta);
          mediaList.appendChild(div);
        });
      }
    });
  } catch(e) {
    console.log("Error loading captured media:", e);
  }
});
