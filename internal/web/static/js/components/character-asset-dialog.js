import { html, useState } from '../lib.js';
import { useLang } from '../i18n.js';
import { api } from '../api.js';

export function CharacterAssetDialog({ character, onClose, onSave, addToast }) {
  const { t } = useLang();
  const isEdit = !!character;

  const [name, setName] = useState(character?.name || '');
  const [bio, setBio] = useState(character?.bio || '');
  const [lore, setLore] = useState(character?.lore || '');
  const [style, setStyle] = useState(character?.style || '');
  const [topics, setTopics] = useState(character?.topics || '');
  const [adjectives, setAdjectives] = useState(character?.adjectives || '');
  const [saving, setSaving] = useState(false);

  const handleSubmit = async (e) => {
    e.preventDefault();
    if (!name.trim()) return;

    setSaving(true);
    try {
      const data = { name: name.trim(), bio, lore, style, topics, adjectives };
      if (isEdit) {
        await api.updateCharacterAsset(character.id, data);
      } else {
        await api.createCharacterAsset(data);
      }
      addToast(isEdit ? t('assets.updated') : t('assets.created'), 'success');
      onSave();
    } catch (err) {
      addToast(err.message, 'error');
    } finally {
      setSaving(false);
    }
  };

  return html`
    <div class="dialog-overlay" onClick=${onClose}>
      <div class="dialog" style="max-width:540px" onClick=${(e) => e.stopPropagation()}>
        <div class="dialog-header">
          <h2>${isEdit ? t('assets.editCharacter') : t('assets.addCharacter')}</h2>
          <button class="dialog-close" onClick=${onClose}>✕</button>
        </div>
        <form onSubmit=${handleSubmit}>
          <div class="dialog-body">
            <label class="form-label">
              ${t('character.name')}
              <input type="text" class="form-input" value=${name} onInput=${(e) => setName(e.target.value)}
                placeholder=${t('character.nameHint')} required autofocus />
            </label>

            <label class="form-label">
              ${t('character.bio')}
              <textarea class="form-input form-textarea" value=${bio} onInput=${(e) => setBio(e.target.value)}
                placeholder=${t('character.bioHint')} rows="3"></textarea>
            </label>

            <label class="form-label">
              ${t('character.lore')}
              <textarea class="form-input form-textarea" value=${lore} onInput=${(e) => setLore(e.target.value)}
                placeholder=${t('character.loreHint')} rows="3"></textarea>
            </label>

            <label class="form-label">
              ${t('character.style')}
              <textarea class="form-input form-textarea" value=${style} onInput=${(e) => setStyle(e.target.value)}
                placeholder=${t('character.styleHint')} rows="2"></textarea>
            </label>

            <label class="form-label">
              ${t('character.topics')}
              <textarea class="form-input form-textarea" value=${topics} onInput=${(e) => setTopics(e.target.value)}
                placeholder=${t('character.topicsHint')} rows="2"></textarea>
            </label>

            <label class="form-label">
              ${t('character.adjectives')}
              <textarea class="form-input form-textarea" value=${adjectives} onInput=${(e) => setAdjectives(e.target.value)}
                placeholder=${t('character.adjectivesHint')} rows="2"></textarea>
            </label>
          </div>
          <div class="dialog-footer">
            <button type="button" class="btn btn-ghost" onClick=${onClose}>${t('configure.cancel')}</button>
            <button type="submit" class="btn btn-primary" disabled=${saving || !name.trim()}>
              ${saving ? t('assets.saving') : t('configure.submit')}
            </button>
          </div>
        </form>
      </div>
    </div>
  `;
}
