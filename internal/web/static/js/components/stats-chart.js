import { html, useEffect, useRef } from '../lib.js';
import { useLang } from '../i18n.js';
import { formatBytes } from '../utils.js';

const HISTORY_LEN = 30;

function Sparkline({ data, max, color, label, valueText }) {
  if (data.length === 0) return null;

  const w = 200, h = 40, pad = 1;
  const n = data.length;
  const effectiveMax = max || Math.max(...data, 1);

  const points = data.map((v, i) => {
    const x = pad + (i / (HISTORY_LEN - 1)) * (w - 2 * pad);
    const y = h - pad - (v / effectiveMax) * (h - 2 * pad);
    return `${x},${y}`;
  });

  const fillPoints = [
    `${pad},${h - pad}`,
    ...points,
    `${pad + ((n - 1) / (HISTORY_LEN - 1)) * (w - 2 * pad)},${h - pad}`,
  ];

  return html`
    <div class="spark-row">
      <span class="spark-label">${label}</span>
      <svg class="spark-svg" viewBox="0 0 ${w} ${h}" preserveAspectRatio="none">
        <polygon points=${fillPoints.join(' ')} fill=${color} opacity="0.15" />
        <polyline points=${points.join(' ')} fill="none" stroke=${color} stroke-width="1.5" />
      </svg>
      <span class="spark-value">${valueText}</span>
    </div>
  `;
}

export function StatsChart({ stats }) {
  const { t } = useLang();
  const cpuHistory = useRef([]);
  const memHistory = useRef([]);

  useEffect(() => {
    if (!stats) return;
    cpuHistory.current = [...cpuHistory.current, stats.cpu_percent].slice(-HISTORY_LEN);
    memHistory.current = [...memHistory.current, stats.memory_usage].slice(-HISTORY_LEN);
  }, [stats]);

  const cpu = stats?.cpu_percent ?? 0;
  const memUsed = stats?.memory_usage ?? 0;
  const memLimit = stats?.memory_limit ?? 1;

  return html`
    <div class="stats-chart">
      <div class="stats-chart-header">${t('stats.title')}</div>
      <${Sparkline}
        data=${cpuHistory.current}
        max=${100}
        color="var(--primary)"
        label="CPU"
        valueText="${cpu.toFixed(1)}%"
      />
      <${Sparkline}
        data=${memHistory.current}
        max=${memLimit}
        color="var(--info)"
        label="MEM"
        valueText="${formatBytes(memUsed)} / ${formatBytes(memLimit)}"
      />
    </div>
  `;
}
