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

// Media Sniffer State
let capturedMedia = {};

function addMediaToTab(tabId, url, ext, title) {
  if (tabId < 0) return;
  if (!capturedMedia[tabId]) capturedMedia[tabId] = [];
  
  if (!capturedMedia[tabId].find(m => m.url === url)) {
    capturedMedia[tabId].push({ url, ext, title, timestamp: Date.now() });
    
    if (chrome.action && chrome.action.setBadgeText) {
      chrome.action.setBadgeText({
        text: capturedMedia[tabId].length.toString(),
        tabId: tabId
      });
      chrome.action.setBadgeBackgroundColor({ color: "#6c5ce7", tabId: tabId });
    }
  }
}

// Sniff network requests for streams
chrome.webRequest.onResponseStarted.addListener(
  (details) => {
    if (details.type === "media" || details.type === "xmlhttprequest") {
      let contentType = "";
      for (let header of details.responseHeaders || []) {
        if (header.name.toLowerCase() === "content-type") {
          contentType = header.value.toLowerCase();
          break;
        }
      }

      const url = details.url.toLowerCase();
      const isM3U8 = url.includes(".m3u8") || contentType.includes("mpegurl");
      const isMPD = url.includes(".mpd") || contentType.includes("dash+xml");
      const isVideo = contentType.startsWith("video/") && !url.includes("googlevideo.com/videoplayback");
      
      if (isM3U8) addMediaToTab(details.tabId, details.url, "m3u8", "HLS Stream");
      else if (isMPD) addMediaToTab(details.tabId, details.url, "mpd", "DASH Stream");
      else if (isVideo) {
        // Simple heuristic for filename
        let name = "Video";
        try { name = new URL(details.url).pathname.split('/').pop() || "Video"; } catch(e){}
        addMediaToTab(details.tabId, details.url, "mp4", name);
      }
    }
  },
  { urls: ["<all_urls>"] },
  ["responseHeaders"]
);

// Tab cleanup and YouTube detection
if (chrome.tabs) {
  chrome.tabs.onUpdated.addListener((tabId, changeInfo, tab) => {
    if (changeInfo.status === 'loading') {
      capturedMedia[tabId] = [];
      if (chrome.action && chrome.action.setBadgeText) {
        chrome.action.setBadgeText({ text: "", tabId: tabId });
      }
    }
    // Detect YouTube Watch pages explicitly
    if (changeInfo.status === 'complete' && tab.url && tab.url.includes("youtube.com/watch")) {
      addMediaToTab(tabId, tab.url, "youtube", tab.title || "YouTube Video");
    }
  });

  chrome.tabs.onRemoved.addListener((tabId) => {
    delete capturedMedia[tabId];
  });
}

// Listen for messages from content script and popup
chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
  if (message.action === "darfin-download-media") {
    if (!isEnabled) {
      sendResponse({ status: "disabled" });
      return;
    }
    sendToDarfin(message.url, message.filename || "", message.referrer || "", null, null);
    sendResponse({ status: "ok" });
    return false;
  }
  
  if (message.action === "get-captured-media") {
    chrome.tabs.query({active: true, currentWindow: true}, function(tabs) {
      if (tabs && tabs.length > 0) {
        const tabId = tabs[0].id;
        sendResponse({ media: capturedMedia[tabId] || [] });
      } else {
        sendResponse({ media: [] });
      }
    });
    return true; // Keep channel open for async response
  }
});
