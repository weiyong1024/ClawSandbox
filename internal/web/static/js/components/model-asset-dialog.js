import { html, useState, useRef, useEffect } from '../lib.js';
import { useLang } from '../i18n.js';
import { api } from '../api.js';

const MODEL_PRESETS = {
  anthropic: ['claude-opus-4-6', 'claude-sonnet-4-6'],
  openai: ['gpt-5.4', 'o3', 'gpt-5-mini'],
  'openai-codex': ['gpt-5.4', 'o3', 'gpt-5-mini'],
  google: ['gemini-3.1-pro-preview', 'gemini-3-flash-preview', 'gemini-2.5-pro'],
  deepseek: ['deepseek-chat', 'deepseek-reasoner'],
};

export function ModelAssetDialog({ model, onClose, onSave, addToast }) {
  const { t } = useLang();
  const isEdit = !!model;
  const isCodex = (model?.provider === 'openai-codex');

  const [name, setName] = useState(model?.name || '');
  const [provider, setProvider] = useState(model?.provider || 'openai');
  const [apiKey, setApiKey] = useState(model?.api_key || '');
  const [selectedModel, setSelectedModel] = useState(model?.model || '');
  const [customModel, setCustomModel] = useState('');
  const [saving, setSaving] = useState(false);
  const [testing, setTesting] = useState(false);
  const [validated, setValidated] = useState(isEdit && model?.validated);

  // Codex OAuth state
  const [codexLogging, setCodexLogging] = useState(false);
  const [codexAccount, setCodexAccount] = useState(model?.oauth_account_id || '');
  const pollRef = useRef(null);

  // Clean up polling on unmount.
  useEffect(() => () => { if (pollRef.current) clearInterval(pollRef.current); }, []);

  const isCodexProvider = provider === 'openai-codex';
  const presets = MODEL_PRESETS[provider] || [];
  const isCustom = selectedModel && !presets.includes(selectedModel);
  const effectiveModel = isCustom ? selectedModel : (selectedModel || presets[0] || '');

  const NAME_HINTS = {
    anthropic: 'e.g. Claude Opus 4.6',
    openai: 'e.g. GPT-5.4 Production',
    'openai-codex': 'e.g. ChatGPT GPT-5.4',
    google: 'e.g. Google AI Studio Gemini',
    deepseek: 'e.g. DeepSeek Chat',
  };

  const handleProviderChange = (newProvider) => {
    setProvider(newProvider);
    setSelectedModel('');
    setCustomModel('');
    setValidated(false);
    setCodexAccount('');
    setCodexLogging(false);
    if (pollRef.current) { clearInterval(pollRef.current); pollRef.current = null; }
  };

  // Standard API key test flow.
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

  // Codex OAuth login flow.
  const handleCodexLogin = async () => {
    const modelToUse = customModel || effectiveModel;
    if (!modelToUse) {
      addToast(t('assets.codexNeedModel'), 'error');
      return;
    }

    setCodexLogging(true);
    setValidated(false);
    setCodexAccount('');

    try {
      const { auth_url, state } = await api.startCodexOAuth(modelToUse, name);

      // Open the auth URL in a popup.
      const popup = window.open(auth_url, 'codex-oauth', 'width=600,height=700,popup=yes');

      // Poll for completion.
      pollRef.current = setInterval(async () => {
        try {
          const poll = await api.pollCodexOAuth(state);
          if (poll.status === 'completed') {
            clearInterval(pollRef.current);
            pollRef.current = null;
            setCodexLogging(false);
            setValidated(true);
            setCodexAccount(poll.account_id || '');
            if (popup && !popup.closed) popup.close();
            addToast(t('assets.codexSuccess'), 'success');
            // Asset was created by the poll endpoint — close dialog and refresh.
            onSave();
          } else if (poll.status === 'failed') {
            clearInterval(pollRef.current);
            pollRef.current = null;
            setCodexLogging(false);
            if (popup && !popup.closed) popup.close();
            addToast(poll.error || t('assets.testFailed'), 'error');
          }
        } catch {
          // Ignore transient poll errors.
        }
      }, 2000);
    } catch (err) {
      setCodexLogging(false);
      addToast(err.message, 'error');
    }
  };

  // Standard submit for API key providers.
  const handleSubmit = async (e) => {
    e.preventDefault();
    if (isCodexProvider) {
      // Codex creates the asset via the poll endpoint. If editing, disallow for now.
      if (isEdit) {
        addToast('Re-authenticate to update Codex credentials', 'error');
      }
      return;
    }
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
                placeholder=${NAME_HINTS[provider] || t('assets.nameHint')} />
            </label>

            <label class="form-label">
              ${t('configure.provider')}
              <select class="form-input" value=${provider} onChange=${(e) => handleProviderChange(e.target.value)}
                disabled=${isEdit && isCodex}>
                <option value="openai">OpenAI ★ ${t('image.recommended')}</option>
                <option value="openai-codex">ChatGPT (Codex) ⚠️</option>
                <option value="anthropic">Anthropic</option>
                <option value="google">Google AI Studio</option>
                <option value="deepseek">DeepSeek</option>
              </select>
            </label>

            ${isCodexProvider && html`
              <p style="margin:8px 0 4px;color:#f59e0b;font-size:0.85rem">
                ⚠️ ${t('assets.codexUnavailable')}
              </p>
            `}

            ${!isCodexProvider && html`
              <label class="form-label">
                ${t('configure.apiKey')}
                <input type="password" class="form-input" value=${apiKey}
                  onInput=${(e) => { setApiKey(e.target.value); setValidated(false); }}
                  required autofocus />
              </label>
            `}

            <label class="form-label">
              ${t('configure.model')}
              <input type="text" class="form-input" list=${`models-${provider}`}
                value=${customModel || selectedModel}
                onInput=${(e) => { setCustomModel(e.target.value); setSelectedModel(e.target.value); setValidated(false); }}
                placeholder=${presets[0] || 'model-name'} required />
              <datalist id=${`models-${provider}`}>
                ${presets.map(m => html`<option key=${m} value=${m} />`)}
              </datalist>
            </label>

            <div style="margin-top: 12px">
              ${isCodexProvider ? html`
                <button type="button" class="btn btn-configure" onClick=${handleCodexLogin}
                  disabled=${codexLogging || !(customModel || effectiveModel)}>
                  ${codexLogging ? t('assets.codexLoggingIn') : t('assets.codexLogin')}
                </button>
                ${validated && html`<span style="margin-left:8px;color:var(--success)">✅ ${t('assets.codexSuccess')}</span>`}
                ${codexAccount && html`<div style="margin-top:6px;color:var(--text-secondary);font-size:0.85rem">
                  ${t('assets.codexAccount')}: ${codexAccount}
                </div>`}
              ` : html`
                <button type="button" class="btn btn-configure" onClick=${handleTest}
                  disabled=${testing || !apiKey || !(customModel || effectiveModel)}>
                  ${testing ? t('assets.testing') : t('assets.test')}
                </button>
                ${validated && html`<span style="margin-left:8px;color:var(--success)">✅ ${t('assets.validated')}</span>`}
              `}
            </div>
          </div>
          <div class="dialog-footer">
            <button type="button" class="btn btn-ghost" onClick=${onClose}>${t('configure.cancel')}</button>
            ${!isCodexProvider && html`
              <button type="submit" class="btn btn-primary" disabled=${saving || !validated}>
                ${saving ? t('assets.saving') : t('configure.submit')}
              </button>
            `}
          </div>
        </form>
      </div>
    </div>
  `;
}
