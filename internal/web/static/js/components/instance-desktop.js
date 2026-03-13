import { html } from '../lib.js';
import { useLang } from '../i18n.js';
import { LogsViewer } from './logs-viewer.js';
import { StatsChart } from './stats-chart.js';

export function InstanceDesktop({ instance, stats, pending, onStart, onStop, onBack }) {
  const { t } = useLang();
  const busy = !!pending;

  if (!instance) {
    return html`
      <div class="desktop-not-found">
        <p>${t('desktop.notFound')}</p>
        <button class="btn btn-ghost" onClick=${onBack}>${t('desktop.back')}</button>
      </div>
    `;
  }

  const isRunning = instance.status === 'running';
  const novncUrl = `${location.protocol}//${location.hostname}:${instance.novnc_port}`;

  return html`
    <div class="instance-desktop">
      <div class="desktop-toolbar">
        <button class="btn btn-ghost" onClick=${onBack}>← ${t('sidebar.instances')}</button>
        <div class="desktop-info">
          <span class="desktop-name">${instance.name}</span>
          <span class="status-badge ${isRunning ? 'status-running' : 'status-stopped'}">
            <span class="status-dot"></span>
            ${pending ? t(`action.${pending}`) : instance.status}
          </span>
        </div>
        <div class="desktop-actions">
          ${isRunning
            ? html`<button class="btn btn-sm btn-warning" disabled=${busy} onClick=${() => onStop(instance.name)}>
                ${pending === 'stopping' ? t('action.stopping') : t('card.suspend')}
              </button>`
            : html`<button class="btn btn-sm btn-success" disabled=${busy} onClick=${() => onStart(instance.name)}>
                ${pending === 'starting' ? t('action.starting') : t('card.resume')}
              </button>`
          }
          <a class="btn btn-sm btn-desktop" href=${novncUrl} target="_blank" rel="noopener">${t('desktop.newTab')}</a>
        </div>
      </div>

      <div class="desktop-body">
        <div class="desktop-main">
          ${isRunning ? html`
            <iframe
              class="desktop-vnc"
              src=${novncUrl}
              allow="clipboard-read; clipboard-write"
            ></iframe>
          ` : html`
            <div class="desktop-vnc-placeholder">
              <div class="desktop-vnc-placeholder-icon">🖥</div>
              <p>${t('desktop.stopped')}</p>
            </div>
          `}
        </div>
        <div class="desktop-sidebar">
          <${StatsChart} stats=${stats} />
          <${LogsViewer} name=${instance.name} />
        </div>
      </div>
    </div>
  `;
}
