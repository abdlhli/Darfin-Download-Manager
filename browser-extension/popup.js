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
});
