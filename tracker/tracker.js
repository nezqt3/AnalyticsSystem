(function () {
  var script = document.currentScript;

  function readAttribute(name, fallback) {
    if (!script) return fallback;
    return script.getAttribute(name) || fallback;
  }

  function uuid() {
    return "xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx".replace(/[xy]/g, function (char) {
      var random = (Math.random() * 16) | 0;
      var value = char === "x" ? random : (random & 0x3) | 0x8;
      return value.toString(16);
    });
  }

  function clamp(value, min, max) {
    return Math.max(min, Math.min(max, value));
  }

  function safeText(value) {
    return (value || "").replace(/\s+/g, " ").trim().slice(0, 120);
  }

  function buildSelector(element) {
    if (!element || element.nodeType !== 1) return "";

    var current = element;
    var parts = [];
    while (current && current.nodeType === 1 && parts.length < 4) {
      var segment = current.tagName.toLowerCase();
      if (current.id) {
        segment += "#" + current.id.slice(0, 40);
        parts.unshift(segment);
        break;
      }
      if (current.classList && current.classList.length) {
        segment += "." + Array.prototype.slice.call(current.classList, 0, 2).join(".");
      }
      parts.unshift(segment);
      current = current.parentElement;
    }
    return parts.join(" > ");
  }

  function extractClickMeta(target) {
    var element = target && target.closest ? target.closest("a, button, input, textarea, select, label, [role='button'], [data-track]") || target : target;
    if (!element || element.nodeType !== 1) {
      return {
        tag: "",
        id: "",
        class: "",
        text: "",
        href: "",
        selector: ""
      };
    }

    return {
      tag: (element.tagName || "").toLowerCase(),
      id: element.id || "",
      class: typeof element.className === "string" ? element.className.slice(0, 120) : "",
      text: safeText(element.innerText || element.textContent || element.value || ""),
      href: element.href || element.getAttribute("href") || "",
      selector: buildSelector(element)
    };
  }

  var endpoint = readAttribute("data-endpoint", "http://localhost:8080/collect");
  var siteId = parseInt(readAttribute("data-site", "0"), 10);
  if (!siteId) {
    console.warn("analytics: data-site is required");
    return;
  }

  var endpointURL = new URL(endpoint, window.location.href);
  var sessionStorageKey = "analytics_session_id";
  var sessionId = localStorage.getItem(sessionStorageKey);
  if (!sessionId) {
    sessionId = uuid();
    localStorage.setItem(sessionStorageKey, sessionId);
  }

  var queue = [];
  var flushTimer = null;
  var maxBatch = 20;
  var flushInterval = 5000;
  var lastTrackedPath = "";
  var maxScrollDepth = 0;

  function getPath() {
    return window.location.pathname || "/";
  }

  function getUTM() {
    var currentURL = new URL(window.location.href);
    return {
      utm_source: currentURL.searchParams.get("utm_source") || "",
      utm_medium: currentURL.searchParams.get("utm_medium") || "",
      utm_campaign: currentURL.searchParams.get("utm_campaign") || ""
    };
  }

  function enqueue(payload) {
    queue.push(payload);
    if (queue.length >= maxBatch) {
      flush();
      return;
    }
    if (!flushTimer) {
      flushTimer = setTimeout(flush, flushInterval);
    }
  }

  function send(eventType, extra) {
    var utm = getUTM();
    var payload = {
      site_id: siteId,
      event_type: eventType,
      path: getPath(),
      title: document.title,
      referrer: document.referrer || "",
      source: utm.utm_source,
      utm_source: utm.utm_source,
      utm_medium: utm.utm_medium,
      utm_campaign: utm.utm_campaign,
      entry_url: window.location.href,
      meta: "",
      screen_w: window.innerWidth,
      screen_h: window.innerHeight,
      session_id: sessionId,
      user_id: ""
    };

    if (extra) {
      for (var key in extra) {
        payload[key] = extra[key];
      }
    }
    if (extra && extra.meta) {
      payload.meta = JSON.stringify(extra.meta);
    }
    enqueue(payload);
  }

  function flush() {
    if (flushTimer) {
      clearTimeout(flushTimer);
      flushTimer = null;
    }
    if (!queue.length) {
      return;
    }

    var batch = queue.splice(0, queue.length);
    var body = JSON.stringify(batch);

    if (navigator.sendBeacon) {
      var blob = new Blob([body], { type: "application/json" });
      if (navigator.sendBeacon(endpointURL.toString(), blob)) {
        return;
      }
    }

    fetch(endpointURL.toString(), {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: body,
      keepalive: true,
      mode: "cors",
      credentials: "omit"
    }).catch(function () {});
  }

  function trackPageview(force) {
    var nextPath = getPath();
    if (!force && nextPath === lastTrackedPath) {
      return;
    }
    lastTrackedPath = nextPath;
    maxScrollDepth = 0;
    send("pageview", {
      meta: {
        location: window.location.href
      }
    });
  }

  function trackNavigation() {
    window.setTimeout(function () {
      trackPageview(false);
    }, 0);
  }

  var originalPushState = history.pushState;
  history.pushState = function () {
    var result = originalPushState.apply(this, arguments);
    trackNavigation();
    return result;
  };

  var originalReplaceState = history.replaceState;
  history.replaceState = function () {
    var result = originalReplaceState.apply(this, arguments);
    trackNavigation();
    return result;
  };

  window.addEventListener("popstate", trackNavigation);
  window.addEventListener("hashchange", trackNavigation);
  window.addEventListener("beforeunload", flush);
  document.addEventListener("visibilitychange", function () {
    if (document.visibilityState === "hidden") {
      flush();
    }
  });

  document.addEventListener("click", function (event) {
    var meta = extractClickMeta(event.target || null);
    send("click", {
      x: event.clientX,
      y: event.clientY,
      meta: meta
    });
  }, true);

  var lastScrollSentAt = 0;
  window.addEventListener("scroll", function () {
    var now = Date.now();
    if (now - lastScrollSentAt < 1500) {
      return;
    }

    var root = document.documentElement;
    var scrollTop = window.pageYOffset || root.scrollTop || 0;
    var viewportHeight = window.innerHeight || 0;
    var totalHeight = Math.max(root.scrollHeight, root.offsetHeight, root.clientHeight);
    if (!totalHeight) {
      return;
    }

    var depth = clamp(Math.round(((scrollTop + viewportHeight) / totalHeight) * 100), 0, 100);
    if (depth <= maxScrollDepth) {
      return;
    }

    maxScrollDepth = depth;
    lastScrollSentAt = now;
    send("scroll", {
      meta: {
        depth_pct: depth
      }
    });
  }, { passive: true });

  document.addEventListener("submit", function (event) {
    var form = event.target;
    send("form_submit", {
      meta: {
        id: form && form.id ? form.id : "",
        action: form && form.action ? form.action : "",
        selector: buildSelector(form)
      }
    });
  }, true);

  trackPageview(true);
})();
