export function connectStats(onMessage, onStatus) {
  return connectWS('/api/v1/ws/stats', onMessage, false, onStatus);
}

export function connectEvents(onMessage) {
  return connectWS('/api/v1/ws/events', onMessage);
}

export function connectLogs(name, onMessage) {
  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
  const url = `${proto}//${location.host}/api/v1/ws/logs/${encodeURIComponent(name)}`;
  return connectWS(url, onMessage, true);
}

function connectWS(pathOrUrl, onMessage, raw = false, onStatus) {
  const isAbsolute = pathOrUrl.startsWith('ws://') || pathOrUrl.startsWith('wss://');
  const url = isAbsolute ? pathOrUrl : `${location.protocol === 'https:' ? 'wss:' : 'ws:'}//${location.host}${pathOrUrl}`;
  let ws;
  let closed = false;
  let timer;
  let retryDelay = 1000;
  const maxDelay = 30000;

  function connect() {
    if (closed) return;
    ws = new WebSocket(url);
    ws.onopen = () => {
      retryDelay = 1000;
      if (onStatus) onStatus(true);
    };
    ws.onmessage = (e) => {
      if (raw) { onMessage(e.data); return; }
      try { onMessage(JSON.parse(e.data)); } catch { /* ignore */ }
    };
    ws.onclose = () => {
      if (onStatus) onStatus(false);
      if (!closed) {
        timer = setTimeout(connect, retryDelay);
        retryDelay = Math.min(retryDelay * 2, maxDelay);
      }
    };
    ws.onerror = () => ws.close();
  }

  connect();

  return {
    close() {
      closed = true;
      clearTimeout(timer);
      if (ws) ws.close();
    },
  };
}
