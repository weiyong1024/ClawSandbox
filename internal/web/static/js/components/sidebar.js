import { html } from '../lib.js';
import { useLang } from '../i18n.js';

export function Sidebar({ currentRoute, onNavigate }) {
  const { t } = useLang();

  const menuGroups = [
    {
      label: t('sidebar.assets'),
      items: [
        { route: '#/assets/models', label: t('sidebar.models'), icon: '🤖' },
        { route: '#/assets/channels', label: t('sidebar.channels'), icon: '💬' },
      ],
    },
    {
      label: t('sidebar.fleet'),
      items: [
        { route: '#/fleet', label: t('sidebar.instances'), icon: '🦞' },
        { route: '#/fleet/snapshots', label: t('sidebar.snapshots'), icon: '📸' },
      ],
    },
    {
      label: t('sidebar.system'),
      items: [
        { route: '#/system/image', label: t('sidebar.image'), icon: '📦' },
      ],
    },
  ];

  return html`
    <nav class="sidebar">
      ${menuGroups.map(group => html`
        <div class="sidebar-group" key=${group.label}>
          <div class="sidebar-group-label">${group.label}</div>
          ${group.items.map(item => html`
            <button
              key=${item.route}
              class="sidebar-item ${currentRoute === item.route ? 'sidebar-item-active' : ''}"
              onClick=${() => onNavigate(item.route)}
            >
              <span class="sidebar-icon">${item.icon}</span>
              <span>${item.label}</span>
            </button>
          `)}
        </div>
      `)}
    </nav>
  `;
}
