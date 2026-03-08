import { html, useState } from '../lib.js';
import { useLang } from '../i18n.js';

export function CreateDialog({ onClose, onCreate }) {
  const { t } = useLang();
  const [count, setCount] = useState(1);
  const [creating, setCreating] = useState(false);

  const handleSubmit = async (e) => {
    e.preventDefault();
    setCreating(true);
    try {
      await onCreate(count);
    } finally {
      setCreating(false);
    }
  };

  return html`
    <div class="dialog-overlay" onClick=${onClose}>
      <div class="dialog" onClick=${(e) => e.stopPropagation()}>
        <div class="dialog-header">
          <h2>${t('create.title')}</h2>
          <button class="dialog-close" onClick=${onClose}>✕</button>
        </div>
        <form onSubmit=${handleSubmit}>
          <div class="dialog-body">
            <label class="form-label">
              ${t('create.label')}
              <input
                type="number"
                class="form-input"
                min="1"
                max="20"
                value=${count}
                onInput=${(e) => setCount(parseInt(e.target.value) || 1)}
                autofocus
              />
            </label>
            <p class="form-hint">${t('create.hint')}</p>
          </div>
          <div class="dialog-footer">
            <button type="button" class="btn btn-ghost" onClick=${onClose}>${t('create.cancel')}</button>
            <button type="submit" class="btn btn-primary" disabled=${creating}>
              ${creating ? t('create.creating') : t('create.submit', count)}
            </button>
          </div>
        </form>
      </div>
    </div>
  `;
}
