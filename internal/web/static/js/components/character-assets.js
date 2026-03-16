import { html, useState, useEffect, useCallback } from '../lib.js';
import { useLang } from '../i18n.js';
import { api } from '../api.js';
import { CharacterAssetDialog } from './character-asset-dialog.js';

export function CharacterAssets({ addToast }) {
  const { t } = useLang();
  const [characters, setCharacters] = useState([]);
  const [loading, setLoading] = useState(true);
  const [showDialog, setShowDialog] = useState(false);
  const [editCharacter, setEditCharacter] = useState(null);

  const refresh = useCallback(async () => {
    try {
      const data = await api.listCharacterAssets();
      setCharacters(data || []);
    } catch (err) {
      addToast(err.message, 'error');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { refresh(); }, [refresh]);

  const handleDelete = async (character) => {
    if (!confirm(t('assets.confirmDelete', character.name))) return;
    try {
      await api.deleteCharacterAsset(character.id);
      addToast(t('assets.deleted', character.name), 'success');
      refresh();
    } catch (err) {
      addToast(err.message, 'error');
    }
  };

  const handleSave = async () => {
    setShowDialog(false);
    setEditCharacter(null);
    refresh();
  };

  const handleEdit = (character) => {
    setEditCharacter(character);
    setShowDialog(true);
  };

  if (loading) {
    return html`<div class="page-content"><div class="dashboard-loading"><p>${t('dashboard.loading')}</p></div></div>`;
  }

  return html`
    <div class="page-content">
      <div class="page-header">
        <h2 class="page-title">${t('sidebar.characters')}</h2>
        <button class="btn btn-primary" onClick=${() => { setEditCharacter(null); setShowDialog(true); }}>
          ${t('assets.addCharacter')}
        </button>
      </div>

      ${characters.length === 0 ? html`
        <div class="assets-empty">
          <div class="assets-empty-icon">🎭</div>
          <h3>${t('assets.noCharacters')}</h3>
          <p>${t('assets.noCharactersDesc')}</p>
        </div>
      ` : html`
        <div class="assets-list">
          ${characters.map(c => html`
            <div class="asset-card" key=${c.id}>
              <div class="asset-card-header">
                <div class="asset-card-name">${c.name}</div>
              </div>
              <div class="asset-card-details">
                ${c.bio && html`
                  <div class="asset-detail">
                    <span class="asset-detail-label">${t('character.bio')}</span>
                    <span class="asset-detail-value">${c.bio.length > 80 ? c.bio.slice(0, 80) + '...' : c.bio}</span>
                  </div>
                `}
                ${c.style && html`
                  <div class="asset-detail">
                    <span class="asset-detail-label">${t('character.style')}</span>
                    <span class="asset-detail-value">${c.style.length > 80 ? c.style.slice(0, 80) + '...' : c.style}</span>
                  </div>
                `}
                ${c.adjectives && html`
                  <div class="asset-detail">
                    <span class="asset-detail-label">${t('character.adjectives')}</span>
                    <span class="asset-detail-value">${c.adjectives.length > 80 ? c.adjectives.slice(0, 80) + '...' : c.adjectives}</span>
                  </div>
                `}
              </div>
              <div class="asset-card-actions">
                <button class="btn btn-sm btn-desktop" onClick=${() => handleEdit(c)}>${t('assets.edit')}</button>
                <button class="btn btn-sm btn-danger" onClick=${() => handleDelete(c)}>
                  ${t('assets.delete')}
                </button>
              </div>
            </div>
          `)}
        </div>
      `}

      ${showDialog && html`
        <${CharacterAssetDialog}
          character=${editCharacter}
          onClose=${() => { setShowDialog(false); setEditCharacter(null); }}
          onSave=${handleSave}
          addToast=${addToast}
        />
      `}
    </div>
  `;
}
