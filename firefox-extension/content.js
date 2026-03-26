// DARFIN Content Script — Media Download Overlay
(function () {
  "use strict";

  // Prevent double-injection
  if (window.__darfinContentLoaded) return;
  window.__darfinContentLoaded = true;

  const MIN_IMAGE_SIZE = 100; // Minimum px dimension to show button
  const HOVER_DELAY = 300; // ms before showing button
  const PROCESSED_ATTR = "data-darfin-processed";

  // SVG download icon
  const DOWNLOAD_ICON = `<svg viewBox="0 0 24 24"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>`;

  let hoverTimeout = null;
  let currentOverlay = null;
  let currentTarget = null;

  /**
   * Check if an element qualifies as downloadable media
   */
  function isDownloadableMedia(el) {
    if (!el || !el.tagName) return false;

    const tag = el.tagName.toLowerCase();

    if (tag === "video") {
      // Must have a direct src or a <source> child
      const src = el.src || el.currentSrc;
      if (src && src.startsWith("http")) return true;
      const source = el.querySelector("source[src]");
      if (source && source.src && source.src.startsWith("http")) return true;
      return false;
    }

    if (tag === "img") {
      const src = el.src || el.currentSrc;
      if (!src || !src.startsWith("http")) return false;
      // Filter tiny images (icons, spacers, etc)
      if (el.naturalWidth > 0 && el.naturalWidth < MIN_IMAGE_SIZE) return false;
      if (el.naturalHeight > 0 && el.naturalHeight < MIN_IMAGE_SIZE) return false;
      // Also check rendered size
      const rect = el.getBoundingClientRect();
      if (rect.width < MIN_IMAGE_SIZE || rect.height < MIN_IMAGE_SIZE) return false;
      // Skip data URIs and inline SVGs
      if (src.startsWith("data:")) return false;
      return true;
    }

    if (tag === "audio") {
      const src = el.src || el.currentSrc;
      if (src && src.startsWith("http")) return true;
      const source = el.querySelector("source[src]");
      if (source && source.src && source.src.startsWith("http")) return true;
      return false;
    }

    return false;
  }

  /**
   * Get the best download URL from a media element
   */
  function getMediaURL(el) {
    const tag = el.tagName.toLowerCase();

    if (tag === "video" || tag === "audio") {
      if (el.src && el.src.startsWith("http")) return el.src;
      if (el.currentSrc && el.currentSrc.startsWith("http")) return el.currentSrc;
      const source = el.querySelector("source[src]");
      if (source) return source.src;
    }

    if (tag === "img") {
      // Prefer srcset highest resolution if available
      return el.src || el.currentSrc;
    }

    return null;
  }

  /**
   * Extract filename from URL
   */
  function getFilenameFromURL(url) {
    try {
      const pathname = new URL(url).pathname;
      const parts = pathname.split("/");
      const last = parts[parts.length - 1];
      if (last && last.includes(".")) {
        return decodeURIComponent(last);
      }
    } catch (e) { }
    return "";
  }

  /**
   * Create the overlay download button
   */
  function createOverlayButton() {
    const btn = document.createElement("button");
    btn.className = "darfin-download-overlay";
    btn.innerHTML = DOWNLOAD_ICON;
    btn.setAttribute("title", "");
    btn.setAttribute("aria-label", "Download with DARFIN");
    return btn;
  }

  /**
   * Position and show the overlay on a media element
   */
  function showOverlay(mediaEl) {
    hideOverlay();

    if (!isDownloadableMedia(mediaEl)) return;

    const url = getMediaURL(mediaEl);
    if (!url) return;

    currentTarget = mediaEl;
    currentOverlay = createOverlayButton();

    // Position overlay relative to the media element
    // We need to find or create a positioned parent
    const parent = mediaEl.parentElement;

    if (parent) {
      const parentStyle = window.getComputedStyle(parent);
      if (parentStyle.position === "static") {
        parent.style.position = "relative";
      }
      parent.appendChild(currentOverlay);
    } else {
      // Fallback: use fixed positioning
      const rect = mediaEl.getBoundingClientRect();
      currentOverlay.style.position = "fixed";
      currentOverlay.style.bottom = "auto";
      currentOverlay.style.right = "auto";
      currentOverlay.style.top = (rect.bottom - 44) + "px";
      currentOverlay.style.left = (rect.right - 44) + "px";
      document.body.appendChild(currentOverlay);
    }

    // Trigger animation
    requestAnimationFrame(() => {
      if (currentOverlay) {
        currentOverlay.classList.add("darfin-visible");
      }
    });

    // Click handler
    currentOverlay.addEventListener("click", (e) => {
      e.preventDefault();
      e.stopPropagation();
      e.stopImmediatePropagation();

      const downloadUrl = getMediaURL(mediaEl);
      if (!downloadUrl) return;

      const filename = getFilenameFromURL(downloadUrl);
      const referrer = window.location.href;

      // Send to background script
      try {
        chrome.runtime.sendMessage({
          action: "darfin-download-media",
          url: downloadUrl,
          filename: filename,
          referrer: referrer
        });
      } catch (err) {
        // Fallback: try browser API (Firefox)
        try {
          browser.runtime.sendMessage({
            action: "darfin-download-media",
            url: downloadUrl,
            filename: filename,
            referrer: referrer
          });
        } catch (e) {
          console.log("DARFIN: Failed to send message", e);
        }
      }

      // Visual feedback
      if (currentOverlay) {
        currentOverlay.classList.add("darfin-success");
        currentOverlay.innerHTML = `<svg viewBox="0 0 24 24"><polyline points="20 6 9 17 4 12"/></svg>`;
        setTimeout(() => hideOverlay(), 1500);
      }
    }, { capture: true });
  }

  /**
   * Hide and remove the overlay
   */
  function hideOverlay() {
    if (currentOverlay) {
      currentOverlay.classList.remove("darfin-visible");
      setTimeout(() => {
        if (currentOverlay && currentOverlay.parentElement) {
          currentOverlay.parentElement.removeChild(currentOverlay);
        }
        currentOverlay = null;
        currentTarget = null;
      }, 200);
    }
  }

  /**
   * Handle mouse enter on media elements
   */
  function handleMouseEnter(e) {
    const el = e.target;
    if (!el || el.classList?.contains("darfin-download-overlay")) return;

    // Walk up if needed (some sites wrap media in containers)
    let mediaEl = null;
    if (isDownloadableMedia(el)) {
      mediaEl = el;
    } else {
      // Check if a child is media
      const child = el.querySelector("video, img, audio");
      if (child && isDownloadableMedia(child)) {
        mediaEl = child;
      }
    }

    if (!mediaEl) return;

    clearTimeout(hoverTimeout);
    hoverTimeout = setTimeout(() => {
      showOverlay(mediaEl);
    }, HOVER_DELAY);
  }

  /**
   * Handle mouse leave on media elements
   */
  function handleMouseLeave(e) {
    clearTimeout(hoverTimeout);

    // Don't hide if we're moving to the overlay button itself
    const relatedTarget = e.relatedTarget;
    if (relatedTarget && (
      relatedTarget.classList?.contains("darfin-download-overlay") ||
      relatedTarget.closest?.(".darfin-download-overlay")
    )) {
      return;
    }

    // Delayed hide to allow moving to the button
    setTimeout(() => {
      if (currentOverlay) {
        const isHoveringOverlay = currentOverlay.matches(":hover");
        if (!isHoveringOverlay) {
          hideOverlay();
        }
      }
    }, 200);
  }

  /**
   * Attach listeners to a media element
   */
  function processElement(el) {
    if (el.hasAttribute(PROCESSED_ATTR)) return;
    el.setAttribute(PROCESSED_ATTR, "true");

    el.addEventListener("mouseenter", handleMouseEnter, { passive: true });
    el.addEventListener("mouseleave", handleMouseLeave, { passive: true });
  }

  /**
   * Scan the DOM for media elements
   */
  function scanDOM(root) {
    const elements = (root || document).querySelectorAll("video, img, audio");
    elements.forEach(processElement);
  }

  /**
   * Observe DOM mutations for dynamically added media
   */
  function observeDOM() {
    const observer = new MutationObserver((mutations) => {
      let shouldScan = false;
      for (const mutation of mutations) {
        for (const node of mutation.addedNodes) {
          if (node.nodeType !== Node.ELEMENT_NODE) continue;

          const tag = node.tagName?.toLowerCase();
          if (tag === "video" || tag === "img" || tag === "audio") {
            processElement(node);
          } else if (node.querySelectorAll) {
            shouldScan = true;
          }
        }
      }
      if (shouldScan) {
        scanDOM();
      }
    });

    observer.observe(document.body, {
      childList: true,
      subtree: true
    });
  }

  // Also add overlay hide when clicking anywhere else
  document.addEventListener("click", (e) => {
    if (currentOverlay && !currentOverlay.contains(e.target) && e.target !== currentTarget) {
      hideOverlay();
    }
  }, true);

  // Handle overlay mouse leave
  document.addEventListener("mouseleave", (e) => {
    if (e.target && e.target.classList?.contains("darfin-download-overlay")) {
      // Check if we moved back to the media element
      const relatedTarget = e.relatedTarget;
      if (relatedTarget !== currentTarget &&
          !currentTarget?.contains(relatedTarget)) {
        hideOverlay();
      }
    }
  }, true);

  // Initialize when DOM is ready
  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", () => {
      scanDOM();
      observeDOM();
    });
  } else {
    scanDOM();
    observeDOM();
  }

  // Re-scan on page navigation (SPA support)
  let lastUrl = location.href;
  new MutationObserver(() => {
    if (location.href !== lastUrl) {
      lastUrl = location.href;
      setTimeout(() => scanDOM(), 1000);
    }
  }).observe(document.body, { childList: true, subtree: true });

})();
