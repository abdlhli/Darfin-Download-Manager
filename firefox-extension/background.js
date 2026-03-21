let isEnabled = true;

// Initialize extension state
chrome.storage.local.get(['isEnabled'], (result) => {
  if (result.isEnabled !== undefined) {
    isEnabled = result.isEnabled;
  }
});

// Update state when changed from popup
chrome.storage.onChanged.addListener((changes, area) => {
  if (area === 'local' && changes.isEnabled) {
    isEnabled = changes.isEnabled.newValue;
  }
});

// Intercept downloads BEFORE "Save As" prompt
chrome.downloads.onDeterminingFilename.addListener((downloadItem, suggest) => {
  if (!isEnabled) {
    suggest();
    return true;
  }
  
  // Ignore downloads that don't look like final files
  if (!downloadItem.url.startsWith('http')) {
    suggest();
    return true;
  }

  // Check if DARFIN app is running
  fetch('http://127.0.0.1:3547/add', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json'
    },
    body: JSON.stringify({
      url: downloadItem.url,
      filename: downloadItem.filename || ''
    })
  })
  .then(response => {
    if (response.ok) {
      // DARFIN received it, cancel browser download
      chrome.downloads.cancel(downloadItem.id);
      console.log("Sent download to DARFIN:", downloadItem.url);
    } else {
      // App error, let browser handle it
      suggest();
    }
  })
  .catch(err => {
    // DARFIN is not running/unreachable, let browser handle it
    suggest();
    console.log("DARFIN closed, continuing in browser.");
  });

  // Return true to indicate we will call suggest() asynchronously
  return true;
});

