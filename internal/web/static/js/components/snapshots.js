import { html, useState, useEffect } from '../lib.js';
import { useLang } from '../i18n.js';
import { api } from '../api.js';
import { formatBytes } from '../utils.js';

export function Snapshots({ addToast }) {
  const { t } = useLang();
  const [snapshots, setSnapshots] = useState([]);
  const [loading, setLoading] = useState(true);

  const refresh = async () => {
    try {
      const data = await api.listSnapshots();
      setSnapshots(data || []);
    } catch (err) {
      addToast(err.message, 'error');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => { refresh(); }, []);

  const handleDelete = async (snap) => {
    if (!confirm(t('snapshot.confirmDelete', snap.name))) return;
    try {
      await api.deleteSnapshot(snap.id);
      addToast(t('snapshot.deleted', snap.name), 'success');
      refresh();
    } catch (err) {
      addToast(err.message, 'error');
    }
  };

  if (loading) return html`<div class="page-loading">${t('dashboard.loading')}</div>`;

  return html`
    <div class="assets-page">
      <div class="assets-header">
        <h2>${t('snapshot.title')}</h2>
      </div>
      ${snapshots.length === 0 ? html`
        <div class="assets-empty">
          <h3>${t('snapshot.noSnapshots')}</h3>
          <p>${t('snapshot.noSnapshotsDesc')}</p>
        </div>
      ` : html`
        <div class="assets-list">
          ${snapshots.map(snap => html`
            <div class="asset-card" key=${snap.id}>
              <div class="asset-info">
                <div class="asset-name">${snap.name}</div>
                <div class="asset-details">
                  ${snap.description ? html`<span>${snap.description}</span><span class="asset-sep">·</span>` : ''}
                  <span>${t('snapshot.source')}: ${snap.source_instance}</span>
                  <span class="asset-sep">·</span>
                  <span>${new Date(snap.created_at).toLocaleString()}</span>
                  <span class="asset-sep">·</span>
                  <span>${formatBytes(snap.size_bytes)}</span>
                </div>
              </div>
              <div class="asset-actions">
                <button class="btn btn-sm btn-danger" onClick=${() => handleDelete(snap)}>
                  ${t('assets.delete')}
                </button>
              </div>
            </div>
          `)}
        </div>
      `}
    </div>
  `;
}
