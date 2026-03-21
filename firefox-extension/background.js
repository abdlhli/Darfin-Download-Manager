let isEnabled = true;

try {
  const storageAPI = (typeof browser !== 'undefined' && browser.storage) ? browser.storage.local : (chrome && chrome.storage ? chrome.storage.local : null);
  const changedAPI = (typeof browser !== 'undefined' && browser.storage) ? browser.storage.onChanged : (chrome && chrome.storage ? chrome.storage.onChanged : null);

  if (storageAPI) {
    storageAPI.get(['isEnabled'], (result) => {
      if (result && result.isEnabled !== undefined) {
        isEnabled = result.isEnabled;
      }
    });
  }

  if (changedAPI) {
    changedAPI.addListener((changes, area) => {
      if (area === 'local' && changes.isEnabled) {
        isEnabled = changes.isEnabled.newValue;
      }
    });
  }
} catch (e) {
  console.log("Storage API initialization bypassed:", e);
}

// Helper to extract cookies, send to DARFIN, and optionally cancel browser download
function sendToDarfin(url, filename, referrer, downloadItemId, suggest) {
  const payload = {
    url: url,
    filename: filename,
    referrer: referrer,
    cookies: ""
  };

  const executeFetch = () => {
    fetch('http://127.0.0.1:3547/add', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload)
    })
    .then(response => {
      if (response.ok) {
        if (downloadItemId !== null && chrome.downloads) {
          chrome.downloads.cancel(downloadItemId, () => {
            let err = chrome.runtime.lastError;
          });
        }
        console.log("Sent download to DARFIN:", url);
      } else {
        if (suggest) suggest();
      }
    })
    .catch(err => {
      console.log("DARFIN unreachable:", err);
      if (suggest) suggest();
    });
  };

  // Try to extract cookies for the target URL
  if (chrome.cookies && chrome.cookies.getAll) {
    try {
      chrome.cookies.getAll({ url: url }, (cookies) => {
        if (cookies && cookies.length > 0) {
          payload.cookies = cookies.map(c => c.name + "=" + c.value).join("; ");
        }
        executeFetch();
      });
    } catch(e) {
      executeFetch();
    }
  } else {
    executeFetch();
  }
}

// Intercept downloads BEFORE "Save As" prompt
chrome.downloads.onDeterminingFilename.addListener((downloadItem, suggest) => {
  if (!isEnabled || !downloadItem.url.startsWith('http')) {
    suggest();
    return true;
  }
  
  sendToDarfin(downloadItem.url, downloadItem.filename || '', downloadItem.referrer || '', downloadItem.id, suggest);
  
  return true;
});

// Setup Context Menu for manual right-click downloads
chrome.runtime.onInstalled.addListener(() => {
  chrome.contextMenus.create({
    id: "download-with-darfin",
    title: "Download with DARFIN",
    contexts: ["link", "video", "audio", "image"]
  });
});

chrome.contextMenus.onClicked.addListener((info, tab) => {
  if (info.menuItemId === "download-with-darfin") {
    const url = info.linkUrl || info.srcUrl;
    if (url) {
      const referrer = info.pageUrl || (tab ? tab.url : "");
      sendToDarfin(url, "", referrer, null, null);
    }
  }
});
