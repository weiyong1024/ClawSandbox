import { html } from '../lib.js';
import { useLang } from '../i18n.js';
import { formatBytes } from '../utils.js';

export function InstanceCard({ instance, stats, pending, selected, onToggleSelect, onStart, onStop, onDestroy, onDesktop, onConsole, onRestartBot, onConfigure, onSnapshot, onSkills, onHermesDashboard }) {
  const { t } = useLang();
  const isRunning = instance.status === 'running';
  const isHermes = instance.runtime_type === 'hermes';
  const cpu = stats?.cpu_percent ?? 0;
  const memUsed = stats?.memory_usage ?? 0;
  const memLimit = stats?.memory_limit ?? 1;
  const memPct = memLimit > 0 ? (memUsed / memLimit) * 100 : 0;
  const busy = !!pending;

  const statusLabel = pending
    ? t(`action.${pending}`)
    : isRunning ? instance.status : t('status.suspended');

  return html`
    <div class="card ${isRunning ? 'card-running' : 'card-stopped'} ${busy ? 'card-busy' : ''} ${selected ? 'card-selected' : ''}">
      <div class="card-header">
        <div class="card-header-left">
          <input type="checkbox" class="card-checkbox"
            checked=${selected}
            onClick=${(e) => { e.stopPropagation(); onToggleSelect(instance.name); }} />
          <div class="card-name">${instance.name}${isHermes
            ? html`<span style="font-size:0.65rem;padding:2px 6px;border-radius:4px;margin-left:8px;background:rgba(76,175,80,0.2);color:#4caf50">☤ Hermes</span>`
            : ''
          }</div>
        </div>
        <span class="status-badge ${isRunning ? 'status-running' : 'status-stopped'}">
          <span class="status-dot"></span>
          ${statusLabel}
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
        ${instance.character_name ? html`
          <div class="config-item">
            <span class="config-label">Character</span>
            <span class="config-value">${instance.character_name}</span>
          </div>
        ` : ''}
        ${instance.channel_name ? html`
          <div class="config-item">
            <span class="config-label">Channel</span>
            <span class="config-value">${instance.channel_name}</span>
          </div>
        ` : ''}
        ${!instance.model_name && !instance.channel_name && !instance.character_name ? html`
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
          ${!isHermes && html`
            <button class="btn btn-sm btn-desktop" onClick=${onDesktop} disabled=${busy}>${t('card.desktop')}</button>
            <button class="btn btn-sm btn-desktop" onClick=${onConsole} disabled=${busy}>${t('card.controlPanel')}</button>
            <button class="btn btn-sm btn-configure" onClick=${onConfigure} disabled=${busy}>
              ${pending === 'configuring' ? t('action.configuring') : t('card.configure')}
            </button>
            <button class="btn btn-sm btn-configure" onClick=${onSkills} disabled=${busy}>
              ${t('card.skills')}
            </button>
            ${instance.model_name && html`
              <button class="btn btn-sm btn-snapshot" onClick=${onSnapshot} disabled=${busy}>
                ${t('card.snapshot')}
              </button>
            `}
            <button class="btn btn-sm btn-reset" onClick=${onRestartBot} disabled=${busy}>
              ${pending === 'restarting' ? t('action.restarting') : t('card.restartBot')}
            </button>
          `}
          ${isHermes && html`
            <button class="btn btn-sm btn-desktop" onClick=${onHermesDashboard} disabled=${busy}>${t('card.hermesDashboard')}</button>
          `}
          <button class="btn btn-sm btn-warning" onClick=${onStop} disabled=${busy}>
            ${pending === 'stopping' ? t('action.stopping') : t('card.suspend')}
          </button>
        ` : html`
          <button class="btn btn-sm btn-success" onClick=${onStart} disabled=${busy}>
            ${pending === 'starting' ? t('action.starting') : t('card.resume')}
          </button>
        `}
        <button class="btn btn-sm btn-danger" onClick=${onDestroy} disabled=${busy}>
          ${pending === 'destroying' ? t('action.destroying') : t('card.destroy')}
        </button>
      </div>
    </div>
  `;
}
