import { html } from '../lib.js';
import { useLang } from '../i18n.js';
import { formatBytes } from '../utils.js';

export function InstanceCard({ instance, stats, pending, onStart, onStop, onDestroy, onDesktop, onConfigure }) {
  const { t } = useLang();
  const isRunning = instance.status === 'running';
  const cpu = stats?.cpu_percent ?? 0;
  const memUsed = stats?.memory_usage ?? 0;
  const memLimit = stats?.memory_limit ?? 1;
  const memPct = memLimit > 0 ? (memUsed / memLimit) * 100 : 0;
  const busy = !!pending;

  return html`
    <div class="card ${isRunning ? 'card-running' : 'card-stopped'} ${busy ? 'card-busy' : ''}">
      <div class="card-header">
        <div class="card-name">${instance.name}</div>
        <span class="status-badge ${isRunning ? 'status-running' : 'status-stopped'}">
          <span class="status-dot"></span>
          ${pending ? t(`action.${pending}`) : instance.status}
        </span>
      </div>

      <div class="card-ports">
        <div class="port-item">
          <span class="port-label">noVNC</span>
          <span class="port-value">${instance.novnc_port}</span>
        </div>
        <div class="port-item">
          <span class="port-label">Gateway</span>
          <span class="port-value">${instance.gateway_port}</span>
        </div>
      </div>

      <div class="card-config">
        ${instance.model_name ? html`
          <div class="config-item">
            <span class="config-label">Model</span>
            <span class="config-value">${instance.model_name}</span>
          </div>
        ` : ''}
        ${instance.channel_name ? html`
          <div class="config-item">
            <span class="config-label">Channel</span>
            <span class="config-value">${instance.channel_name}</span>
          </div>
        ` : ''}
        ${!instance.model_name && !instance.channel_name ? html`
          <div class="config-item">
            <span class="config-value config-unconfigured">${t('card.unconfigured')}</span>
          </div>
        ` : ''}
      </div>

      ${isRunning && stats && html`
        <div class="card-stats">
          <div class="stat-row">
            <span class="stat-label">CPU</span>
            <div class="stat-bar">
              <div class="stat-fill stat-cpu" style="width: ${Math.min(cpu, 100)}%"></div>
            </div>
            <span class="stat-value">${cpu.toFixed(1)}%</span>
          </div>
          <div class="stat-row">
            <span class="stat-label">MEM</span>
            <div class="stat-bar">
              <div class="stat-fill stat-mem" style="width: ${Math.min(memPct, 100)}%"></div>
            </div>
            <span class="stat-value">${formatBytes(memUsed)}</span>
          </div>
        </div>
      `}

      <div class="card-actions">
        ${isRunning ? html`
          <button class="btn btn-sm btn-desktop" onClick=${onDesktop} disabled=${busy}>${t('card.desktop')}</button>
          <button class="btn btn-sm btn-configure" onClick=${onConfigure} disabled=${busy}>
            ${pending === 'configuring' ? t('action.configuring') : t('card.configure')}
          </button>
          <button class="btn btn-sm btn-warning" onClick=${onStop} disabled=${busy}>
            ${pending === 'stopping' ? t('action.stopping') : t('card.stop')}
          </button>
        ` : html`
          <button class="btn btn-sm btn-success" onClick=${onStart} disabled=${busy}>
            ${pending === 'starting' ? t('action.starting') : t('card.start')}
          </button>
        `}
        <button class="btn btn-sm btn-danger" onClick=${onDestroy} disabled=${busy}>
          ${pending === 'destroying' ? t('action.destroying') : t('card.destroy')}
        </button>
      </div>
    </div>
  `;
}
