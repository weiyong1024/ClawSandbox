import { html, useState } from '../lib.js';
import { useLang } from '../i18n.js';
import { api } from '../api.js';

export function SnapshotDialog({ instanceName, onClose, addToast }) {
  const { t } = useLang();
  const defaultName = `${instanceName}-snap-${new Date().toISOString().slice(0, 19).replace(/[T:]/g, '-').replace(/-/g, '').slice(0, 15)}`;
  const [name, setName] = useState(defaultName);
  const [description, setDescription] = useState('');
  const [saving, setSaving] = useState(false);

  const handleSubmit = async (e) => {
    e.preventDefault();
    if (!name.trim()) return;
    setSaving(true);
    try {
      await api.createSnapshot({ instance_name: instanceName, name: name.trim(), description: description.trim() });
      addToast(t('snapshot.saved', name.trim()), 'success');
      onClose();
    } catch (err) {
      addToast(err.message, 'error');
    } finally {
      setSaving(false);
    }
  };

  return html`
    <div class="dialog-overlay" onClick=${onClose}>
      <div class="dialog" onClick=${(e) => e.stopPropagation()}>
        <div class="dialog-header">
          <h2>${t('snapshot.saveTitle')}</h2>
          <button class="dialog-close" onClick=${onClose}>✕</button>
        </div>
        <form onSubmit=${handleSubmit}>
          <div class="dialog-body">
            <label class="form-label">
              ${t('snapshot.name')}
              <input
                type="text"
                class="form-input"
                value=${name}
                onInput=${(e) => setName(e.target.value)}
                autofocus
              />
            </label>
            <label class="form-label">
              ${t('snapshot.description')}
              <input
                type="text"
                class="form-input"
                value=${description}
                onInput=${(e) => setDescription(e.target.value)}
                placeholder=${t('snapshot.descriptionHint')}
              />
            </label>
            <p class="form-hint">${t('snapshot.saveHint')}</p>
          </div>
          <div class="dialog-footer">
            <button type="button" class="btn btn-ghost" onClick=${onClose}>${t('create.cancel')}</button>
            <button type="submit" class="btn btn-primary" disabled=${saving || !name.trim()}>
              ${saving ? t('snapshot.saving') : t('snapshot.save')}
            </button>
          </div>
        </form>
      </div>
    </div>
  `;
}
