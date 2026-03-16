(function () {
  function uuid() {
    return "xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx".replace(/[xy]/g, function (c) {
      var r = (Math.random() * 16) | 0;
      var v = c === "x" ? r : (r & 0x3) | 0x8;
      return v.toString(16);
    });
  }

  var endpoint = (document.currentScript && document.currentScript.getAttribute("data-endpoint")) || "http://localhost:8080/collect";
  var siteIdAttr = document.currentScript && document.currentScript.getAttribute("data-site");
  var siteId = siteIdAttr ? parseInt(siteIdAttr, 10) : 0;
  var endpointURL = new URL(endpoint, window.location.href);
  var sameOrigin = endpointURL.origin === window.location.origin;

  if (!siteId) {
    console.warn("analytics: data-site is required");
    return;
  }

  var sessionKey = "analytics_session_id";
  var sessionId = localStorage.getItem(sessionKey);
  if (!sessionId) {
    sessionId = uuid();
    localStorage.setItem(sessionKey, sessionId);
  }

  function getUtm() {
    var url = new URL(window.location.href);
    return {
      utm_source: url.searchParams.get("utm_source") || "",
      utm_medium: url.searchParams.get("utm_medium") || "",
      utm_campaign: url.searchParams.get("utm_campaign") || ""
    };
  }

  function send(eventType, extra) {
    var utm = getUtm();
    var payload = {
      site_id: siteId,
      event_type: eventType,
      path: window.location.pathname,
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
      user_id: "",
    };
    if (extra) {
      for (var k in extra) payload[k] = extra[k];
    }
    if (extra && extra.meta) {
      payload.meta = JSON.stringify(extra.meta);
    }
    enqueue(payload);
  }

  var queue = [];
  var maxBatch = 20;
  var flushInterval = 5000;
  var flushTimer = null;

  function enqueue(payload) {
    queue.push(payload);
    if (queue.length >= maxBatch) {
      flush();
    } else if (!flushTimer) {
      flushTimer = setTimeout(flush, flushInterval);
    }
  }

  function flush() {
    if (flushTimer) {
      clearTimeout(flushTimer);
      flushTimer = null;
    }
    if (queue.length === 0) return;

    var batch = queue.splice(0, queue.length);
    var body = JSON.stringify(batch);
    if (sameOrigin && navigator.sendBeacon) {
      navigator.sendBeacon(endpoint, new Blob([body], { type: "application/json" }));
    } else {
      fetch(endpoint, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: body,
        keepalive: true,
        mode: "cors",
        credentials: "omit"
      });
    }
  }

  window.addEventListener("beforeunload", flush);
  document.addEventListener("visibilitychange", function () {
    if (document.visibilityState === "hidden") flush();
  });

  send("pageview");

  document.addEventListener("click", function (e) {
    var x = e.clientX;
    var y = e.clientY;
    var t = e.target || {};
    send("click", {
      x: x,
      y: y,
      meta: {
        tag: (t.tagName || "").toLowerCase(),
        id: t.id || "",
        class: (t.className || "").toString().slice(0, 120)
      }
    });
  });

  var lastScroll = 0;
  var maxDepth = 0;
  window.addEventListener("scroll", function () {
    var now = Date.now();
    if (now - lastScroll < 2000) return;
    lastScroll = now;

    var doc = document.documentElement;
    var scrollTop = window.pageYOffset || doc.scrollTop || 0;
    var viewport = window.innerHeight || 0;
    var height = Math.max(doc.scrollHeight, doc.offsetHeight, doc.clientHeight);
    if (height <= 0) return;
    var depth = Math.min(100, Math.round(((scrollTop + viewport) / height) * 100));
    if (depth > maxDepth) {
      maxDepth = depth;
      send("scroll", { meta: { depth_pct: depth } });
    }
  });

  document.addEventListener("submit", function (e) {
    var f = e.target || {};
    send("form_submit", {
      meta: {
        id: f.id || "",
        action: f.action || ""
      }
    });
  });
})();
