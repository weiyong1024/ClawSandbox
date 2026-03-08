import { html, useEffect, useRef, useState } from '../lib.js';
import { useLang } from '../i18n.js';
import { connectLogs } from '../ws.js';

const MAX_LINES = 500;

let lineIdCounter = 0;

export function LogsViewer({ name }) {
  const { t } = useLang();
  const [lines, setLines] = useState([]);
  const containerRef = useRef(null);
  const autoScroll = useRef(true);

  useEffect(() => {
    setLines([]);

    const conn = connectLogs(name, (text) => {
      const newLines = text.split('\n').filter(l => l.length > 0).map(l => ({
        id: ++lineIdCounter,
        text: l,
      }));
      setLines(prev => {
        const combined = [...prev, ...newLines];
        return combined.length > MAX_LINES ? combined.slice(-MAX_LINES) : combined;
      });
    });

    return () => conn.close();
  }, [name]);

  useEffect(() => {
    const el = containerRef.current;
    if (el && autoScroll.current) {
      el.scrollTop = el.scrollHeight;
    }
  }, [lines]);

  const handleScroll = () => {
    const el = containerRef.current;
    if (!el) return;
    autoScroll.current = el.scrollTop + el.clientHeight >= el.scrollHeight - 20;
  };

  return html`
    <div class="logs-viewer">
      <div class="logs-header">
        <span class="logs-title">${t('logs.title')}</span>
        <span class="logs-count">${t('logs.lines', lines.length)}</span>
      </div>
      <div class="logs-body" ref=${containerRef} onScroll=${handleScroll}>
        ${lines.length === 0
          ? html`<div class="logs-empty">${t('logs.waiting')}</div>`
          : lines.map(line => html`<div key=${line.id} class="logs-line">${line.text}</div>`)}
      </div>
    </div>
  `;
}
