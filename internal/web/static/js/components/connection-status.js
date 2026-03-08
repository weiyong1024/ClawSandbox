import { html } from '../lib.js';
import { useLang } from '../i18n.js';

export function ConnectionStatus({ connected }) {
  if (connected) return null;

  const { t } = useLang();
  return html`
    <div class="conn-banner">
      <span class="conn-dot"></span>
      ${t('status.disconnected')}
    </div>
  `;
}
