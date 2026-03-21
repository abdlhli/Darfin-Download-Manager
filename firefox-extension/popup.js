document.addEventListener('DOMContentLoaded', () => {
  const toggleCapture = document.getElementById('toggle-capture');

  // Load current state
  chrome.storage.local.get(['isEnabled'], (result) => {
    if (result.isEnabled !== undefined) {
      toggleCapture.checked = result.isEnabled;
    }
  });

  // Save state on change
  toggleCapture.addEventListener('change', (e) => {
    chrome.storage.local.set({ isEnabled: e.target.checked });
  });
});
