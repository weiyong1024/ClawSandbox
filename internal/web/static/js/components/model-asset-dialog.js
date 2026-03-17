import { html, useState } from '../lib.js';
import { useLang } from '../i18n.js';
import { api } from '../api.js';

const MODEL_PRESETS = {
  anthropic: ['claude-opus-4-6', 'claude-sonnet-4-6'],
  openai: ['gpt-5.4', 'o3', 'gpt-5-mini'],
  google: ['gemini-3.1-pro-preview', 'gemini-3-flash-preview', 'gemini-2.5-pro'],
  deepseek: ['deepseek-chat', 'deepseek-reasoner'],
};

export function ModelAssetDialog({ model, onClose, onSave, addToast }) {
  const { t } = useLang();
  const isEdit = !!model;

  const [name, setName] = useState(model?.name || '');
  const [provider, setProvider] = useState(model?.provider || 'anthropic');
  const [apiKey, setApiKey] = useState(model?.api_key || '');
  const [selectedModel, setSelectedModel] = useState(model?.model || '');
  const [customModel, setCustomModel] = useState('');
  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState(false);
  const [validated, setValidated] = useState(isEdit && model?.validated);

  const presets = MODEL_PRESETS[provider] || [];
  const isCustom = selectedModel && !presets.includes(selectedModel);
  const effectiveModel = isCustom ? selectedModel : (selectedModel || presets[0] || '');

  const handleProviderChange = (newProvider) => {
    setProvider(newProvider);
    setSelectedModel('');
    setCustomModel('');
    setValidated(false);
  };

  const handleTest = async () => {
    const modelToTest = customModel || effectiveModel;
    if (!apiKey || !modelToTest) return;

    setTesting(true);
    try {
      const result = await api.testModelAsset({ provider, api_key: apiKey, model: modelToTest });
      if (result.valid) {
        setValidated(true);
        addToast(t('assets.testSuccess'), 'success');
      } else {
        setValidated(false);
        addToast(result.error || t('assets.testFailed'), 'error');
      }
    } catch (err) {
      setValidated(false);
      addToast(err.message, 'error');
    } finally {
      setTesting(false);
    }
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    if (!validated) {
      addToast(t('assets.mustValidate'), 'error');
      return;
    }

    const finalModel = customModel || effectiveModel;
    setSaving(true);
    try {
      if (isEdit) {
        await api.updateModelAsset(model.id, { name, provider, api_key: apiKey, model: finalModel });
      } else {
        await api.createModelAsset({ name, provider, api_key: apiKey, model: finalModel });
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
      <div class="dialog" style="max-width:480px" onClick=${(e) => e.stopPropagation()}>
        <div class="dialog-header">
          <h2>${isEdit ? t('assets.editModel') : t('assets.addModel')}</h2>
          <button class="dialog-close" onClick=${onClose}>✕</button>
        </div>
        <form onSubmit=${handleSubmit}>
          <div class="dialog-body">
            <label class="form-label">
              ${t('assets.name')}
              <input type="text" class="form-input" value=${name} onInput=${(e) => setName(e.target.value)}
                placeholder=${t('assets.nameHint')} />
            </label>

            <label class="form-label">
              ${t('configure.provider')}
              <select class="form-input" value=${provider} onChange=${(e) => handleProviderChange(e.target.value)}>
                <option value="anthropic">Anthropic</option>
                <option value="openai">OpenAI</option>
                <option value="google">Google</option>
                <option value="deepseek">DeepSeek</option>
              </select>
            </label>

            <label class="form-label">
              ${t('configure.apiKey')}
              <input type="password" class="form-input" value=${apiKey}
                onInput=${(e) => { setApiKey(e.target.value); setValidated(false); }}
                required autofocus />
            </label>

            <label class="form-label">
              ${t('configure.model')}
              <select class="form-input" value=${isCustom ? '__custom__' : selectedModel}
                onChange=${(e) => {
                  if (e.target.value === '__custom__') {
                    setSelectedModel('');
                    setCustomModel('');
                  } else {
                    setSelectedModel(e.target.value);
                    setCustomModel('');
                  }
                  setValidated(false);
                }}>
                ${presets.map(m => html`<option key=${m} value=${m}>${m}</option>`)}
                <option value="__custom__">${t('assets.customModel')}</option>
              </select>
            </label>

            ${(isCustom || (!selectedModel && !presets.length)) && html`
              <label class="form-label">
                ${t('assets.customModelName')}
                <input type="text" class="form-input" value=${customModel || selectedModel}
                  onInput=${(e) => { setCustomModel(e.target.value); setSelectedModel(e.target.value); setValidated(false); }}
                  placeholder="model-name" required />
              </label>
            `}

            <div style="margin-top: 12px">
              <button type="button" class="btn btn-configure" onClick=${handleTest}
                disabled=${testing || !apiKey || !(customModel || effectiveModel)}>
                ${testing ? t('assets.testing') : t('assets.test')}
              </button>
              ${validated && html`<span style="margin-left:8px;color:var(--success)">✅ ${t('assets.validated')}</span>`}
            </div>
          </div>
          <div class="dialog-footer">
            <button type="button" class="btn btn-ghost" onClick=${onClose}>${t('configure.cancel')}</button>
            <button type="submit" class="btn btn-primary" disabled=${saving || !validated}>
              ${saving ? t('assets.saving') : t('configure.submit')}
            </button>
          </div>
        </form>
      </div>
    </div>
  `;
}
