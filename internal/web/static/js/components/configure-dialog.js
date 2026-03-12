import { html, useState, useEffect } from '../lib.js';
import { useLang } from '../i18n.js';
import { api } from '../api.js';

export function ConfigureDialog({ instanceName, currentModelAssetId, onClose, onConfigure }) {
  const { t } = useLang();
  const [models, setModels] = useState([]);
  const [channels, setChannels] = useState([]);
  const [selectedModel, setSelectedModel] = useState(currentModelAssetId || '');
  const [selectedChannel, setSelectedChannel] = useState('');
  const [configuring, setConfiguring] = useState(false);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    Promise.all([
      api.listModelAssets(),
      api.listChannelAssets(),
    ]).then(([modelData, channelData]) => {
      setModels(modelData || []);
      setChannels(channelData || []);

      // Pre-select model assigned to this instance
      if (currentModelAssetId) setSelectedModel(currentModelAssetId);

      // Pre-select channel assigned to this instance
      const assignedCh = (channelData || []).find(c => c.used_by === instanceName);
      if (assignedCh) setSelectedChannel(assignedCh.id);
    }).catch((err) => { console.error('Failed to load assets:', err); }).finally(() => setLoading(false));
  }, [instanceName]);

  const handleSubmit = async (e) => {
    e.preventDefault();
    if (!selectedModel) return;

    setConfiguring(true);
    await onConfigure(instanceName, {
      model_asset_id: selectedModel,
      channel_asset_id: selectedChannel || undefined,
    });
    setConfiguring(false);
  };

  // Models are shared — show all validated models
  const availableModels = models.filter(m => m.validated);
  const availableChannels = channels.filter(c => c.validated && (!c.used_by || c.used_by === instanceName));

  const providerLabel = (p) => {
    const map = { anthropic: 'Anthropic', openai: 'OpenAI', google: 'Google', deepseek: 'DeepSeek' };
    return map[p] || p;
  };

  return html`
    <div class="dialog-overlay" onClick=${onClose}>
      <div class="dialog" style="max-width:500px" onClick=${(e) => e.stopPropagation()}>
        <div class="dialog-header">
          <h2>${t('configure.title')}: ${instanceName}</h2>
          <button class="dialog-close" onClick=${onClose}>✕</button>
        </div>
        ${loading ? html`
          <div class="dialog-body"><p>${t('dashboard.loading')}</p></div>
        ` : html`
          <form onSubmit=${handleSubmit}>
            <div class="dialog-body">
              <div class="form-label">${t('configure.modelConfig')}</div>
              ${availableModels.length === 0 ? html`
                <p class="form-hint">${t('configure.noModels')}</p>
              ` : html`
                <div class="config-select-list">
                  ${availableModels.map(m => html`
                    <label class="config-select-item ${selectedModel === m.id ? 'config-select-active' : ''}" key=${m.id}>
                      <input type="radio" name="model" value=${m.id}
                        checked=${selectedModel === m.id}
                        onChange=${() => setSelectedModel(m.id)} />
                      <div class="config-select-info">
                        <div class="config-select-name">${m.name}</div>
                        <div class="config-select-meta">${providerLabel(m.provider)} / ${m.model}</div>
                      </div>
                      <span class="config-select-badge">✅</span>
                    </label>
                  `)}
                </div>
              `}

              <div class="form-label" style="margin-top:16px">${t('configure.channelConfig')}</div>
              <div class="config-select-list">
                <label class="config-select-item ${!selectedChannel ? 'config-select-active' : ''}">
                  <input type="radio" name="channel" value=""
                    checked=${!selectedChannel}
                    onChange=${() => setSelectedChannel('')} />
                  <div class="config-select-info">
                    <div class="config-select-name">${t('configure.noChannel')}</div>
                  </div>
                </label>
                ${availableChannels.map(c => html`
                  <label class="config-select-item ${selectedChannel === c.id ? 'config-select-active' : ''}" key=${c.id}>
                    <input type="radio" name="channel" value=${c.id}
                      checked=${selectedChannel === c.id}
                      onChange=${() => setSelectedChannel(c.id)} />
                    <div class="config-select-info">
                      <div class="config-select-name">${c.name}</div>
                      <div class="config-select-meta">${c.channel}</div>
                    </div>
                    <span class="config-select-badge">✅</span>
                  </label>
                `)}
              </div>
            </div>
            <div class="dialog-footer">
              ${configuring && html`<span class="form-hint" style="margin-right:auto">${t('configure.timeHint')}</span>`}
              <button type="button" class="btn btn-ghost" onClick=${onClose} disabled=${configuring}>${t('configure.cancel')}</button>
              <button type="submit" class="btn btn-primary" disabled=${configuring || !selectedModel}>
                ${configuring ? t('configure.configuring') : t('configure.submit')}
              </button>
            </div>
          </form>
        `}
      </div>
    </div>
  `;
}
