export function connectStats(onMessage, onStatus) {
  return connectWS('/api/v1/ws/stats', onMessage, false, onStatus);
}

export function connectEvents(onMessage) {
  return connectWS('/api/v1/ws/events', onMessage);
}

export function connectLogs(name, onMessage) {
  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
  const url = `${proto}//${location.host}/api/v1/ws/logs/${name}`;
  let ws;
  let closed = false;
  let timer;

  function connect() {
    if (closed) return;
    ws = new WebSocket(url);
    ws.onmessage = (e) => onMessage(e.data);
    ws.onclose = () => {
      if (!closed) timer = setTimeout(connect, 3000);
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

function connectWS(path, onMessage, raw = false, onStatus) {
  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
  const url = `${proto}//${location.host}${path}`;
  let ws;
  let closed = false;
  let timer;

  function connect() {
    if (closed) return;
    ws = new WebSocket(url);
    ws.onopen = () => { if (onStatus) onStatus(true); };
    ws.onmessage = (e) => {
      if (raw) { onMessage(e.data); return; }
      try { onMessage(JSON.parse(e.data)); } catch { /* ignore */ }
    };
    ws.onclose = () => {
      if (onStatus) onStatus(false);
      if (!closed) timer = setTimeout(connect, 3000);
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
