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

export function Dashboard({ instances, stats, loading, pending, onStart, onStop, onDestroy, onDesktop }) {
  const { t } = useLang();

  if (loading) {
    return html`
      <div class="dashboard-grid">
        <${SkeletonCard} /><${SkeletonCard} /><${SkeletonCard} />
      </div>
    `;
  }

  if (instances.length === 0) {
    return html`
      <div class="dashboard-empty">
        <div class="dashboard-empty-icon">🦞</div>
        <h2>${t('dashboard.empty.title')}</h2>
        <p>${t('dashboard.empty.desc')}</p>
      </div>
    `;
  }

  return html`
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
        />
      `)}
    </div>
  `;
}
