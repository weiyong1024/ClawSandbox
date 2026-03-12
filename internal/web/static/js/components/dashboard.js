import { html } from '../lib.js';
import { useLang } from '../i18n.js';
import { InstanceCard } from './instance-card.js';

function SkeletonCard() {
  return html`
    <div class="card skeleton-card">
      <div class="skeleton-line skeleton-w60"></div>
      <div class="skeleton-line skeleton-w40"></div>
      <div class="skeleton-line skeleton-w80"></div>
      <div class="skeleton-line skeleton-w50"></div>
    </div>
  `;
}

export function Dashboard({ instances, stats, loading, pending, onStart, onStop, onDestroy, onDesktop, onConfigure, onReset, onCreateClick }) {
  const { t } = useLang();

  if (loading) {
    return html`
      <div class="page-content">
        <div class="dashboard-grid">
          <${SkeletonCard} /><${SkeletonCard} /><${SkeletonCard} />
        </div>
      </div>
    `;
  }

  return html`
    <div class="page-content">
      <div class="page-header">
        <h2 class="page-title">${t('sidebar.instances')} <span class="toolbar-count">${t('toolbar.instances', instances.length)}</span></h2>
        <button class="btn btn-primary" onClick=${onCreateClick}>
          ${t('toolbar.create')}
        </button>
      </div>

      ${instances.length === 0 ? html`
        <div class="dashboard-empty">
          <div class="dashboard-empty-icon">🦞</div>
          <h2>${t('dashboard.empty.title')}</h2>
          <p>${t('dashboard.empty.desc')}</p>
        </div>
      ` : html`
        <div class="dashboard-grid">
          ${instances.map(inst => html`
            <${InstanceCard}
              key=${inst.name}
              instance=${inst}
              stats=${stats[inst.name]}
              pending=${pending[inst.name]}
              onStart=${() => onStart(inst.name)}
              onStop=${() => onStop(inst.name)}
              onDestroy=${() => onDestroy(inst.name)}
              onDesktop=${() => onDesktop(inst.name)}
              onConfigure=${() => onConfigure(inst.name)}
              onReset=${() => onReset(inst.name)}
            />
          `)}
        </div>
      `}
    </div>
  `;
}
