import { html, useState, useEffect, useCallback } from '../lib.js';
import { useLang } from '../i18n.js';
import { api } from '../api.js';

const IMAGE_FLAVORS = [
  {
    id: 'lightweight',
    name: 'Lightweight',
    base: 'node:22-bookworm (Debian)',
    size: '~1.4 GB',
    desktop: 'XFCE4',
    available: true,
    recommended: true,
  },
  {
    id: 'full-desktop',
    name: 'Full Desktop',
    base: 'ubuntu:24.04',
    size: '~2.5 GB',
    desktop: 'XFCE4 Full',
    available: false,
    recommended: false,
  },
];

export function ImagePage({ addToast }) {
  const { t } = useLang();
  const [imageStatus, setImageStatus] = useState(null);
  const [loading, setLoading] = useState(true);
  const [building, setBuilding] = useState(false);
  const [buildLogs, setBuildLogs] = useState([]);
  const [pulling, setPulling] = useState(false);
  const [pullLogs, setPullLogs] = useState([]);
  const [selectedFlavor, setSelectedFlavor] = useState('lightweight');
  const [versions, setVersions] = useState(null);
  const [selectedVersion, setSelectedVersion] = useState('');
  const [versionsLoading, setVersionsLoading] = useState(true);

  const checkStatus = useCallback(async () => {
    try {
      const data = await api.imageStatus();
      setImageStatus(data);
    } catch (err) {
      addToast(err.message, 'error');
    } finally {
      setLoading(false);
    }
  }, []);

  const loadVersions = useCallback(async () => {
    setVersionsLoading(true);
    try {
      const data = await api.openclawVersions();
      setVersions(data);
      setSelectedVersion(data.recommended);
      if (data.error) addToast(t('image.versionError'), 'warning');
    } catch {
      setVersions({ recommended: '', versions: [], latest: '' });
      addToast(t('image.versionError'), 'error');
    } finally {
      setVersionsLoading(false);
    }
  }, []);

  useEffect(() => { checkStatus(); loadVersions(); }, [checkStatus, loadVersions]);

  const readSSE = async (endpoint, setLogs, successKey, failKey, fetchInit = {}) => {
    const proto = location.protocol === 'https:' ? 'https:' : 'http:';
    const url = `${proto}//${location.host}${endpoint}`;
    const response = await fetch(url, { method: 'POST', ...fetchInit });
    const reader = response.body.getReader();
    const decoder = new TextDecoder();
    let buffer = '';

    while (true) {
      const { done, value } = await reader.read();
      if (done) break;

      buffer += decoder.decode(value, { stream: true });
      const lines = buffer.split('\n');
      buffer = lines.pop();

      for (const line of lines) {
        if (line.startsWith('data: ')) {
          const msg = line.slice(6);
          setLogs(prev => [...prev, msg]);
        }
        if (line.startsWith('event: done')) {
          addToast(t(successKey), 'success');
        }
        if (line.startsWith('event: error')) {
          addToast(t(failKey), 'error');
        }
      }
    }
  };

  // "恢复默认版本" — pulls the pre-built release image
  const handleResetDefault = async () => {
    setPulling(true);
    setPullLogs([]);
    try {
      await readSSE('/api/v1/image/pull', setPullLogs, 'image.resetSuccess', 'image.resetFailed');
    } catch (err) {
      addToast(err.message, 'error');
    } finally {
      setPulling(false);
      checkStatus();
    }
  };

  // "装载此版本" — builds image with the selected OpenClaw version
  const handleLoadVersion = async () => {
    setBuilding(true);
    setBuildLogs([]);
    try {
      await readSSE('/api/v1/image/build', setBuildLogs, 'image.loadSuccess', 'image.loadFailed', {
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ openclaw_version: selectedVersion }),
      });
    } catch (err) {
      addToast(err.message, 'error');
    } finally {
      setBuilding(false);
      checkStatus();
    }
  };

  const busy = building || pulling;
  const isRecommended = selectedVersion === versions?.recommended;

  if (loading) {
    return html`<div class="page-content"><div class="dashboard-loading"><p>${t('dashboard.loading')}</p></div></div>`;
  }

  return html`
    <div class="page-content">
      <div class="page-header">
        <h2 class="page-title">${t('sidebar.image')}</h2>
      </div>

      <div class="image-section">
        <h3 class="section-title">${t('image.selectFlavor')}</h3>
        <div class="image-flavors">
          ${IMAGE_FLAVORS.map(flavor => html`
            <div
              key=${flavor.id}
              class="image-flavor-card ${selectedFlavor === flavor.id ? 'image-flavor-selected' : ''} ${!flavor.available ? 'image-flavor-disabled' : ''}"
              onClick=${() => flavor.available && setSelectedFlavor(flavor.id)}
            >
              <div class="image-flavor-header">
                <div class="image-flavor-radio">
                  ${selectedFlavor === flavor.id ? '●' : '○'}
                </div>
                <div class="image-flavor-name">
                  ${flavor.name}
                  ${flavor.recommended ? html` <span class="image-flavor-badge">${t('image.recommended')}</span>` : ''}
                  ${!flavor.available ? html` <span class="image-flavor-badge-soon">Coming Soon</span>` : ''}
                </div>
              </div>
              <div class="image-flavor-details">
                <div class="image-flavor-detail">${t('image.baseImage')}: ${flavor.base}</div>
                <div class="image-flavor-detail">${t('image.size')}: ${flavor.size}</div>
                <div class="image-flavor-detail">${t('image.desktop')}: ${flavor.desktop}</div>
              </div>
            </div>
          `)}
        </div>
      </div>

      <div class="image-section">
        <h3 class="section-title">${t('image.openclawVersion')}</h3>
        <div style="display:flex;align-items:center;gap:12px;flex-wrap:wrap">
          ${versionsLoading
            ? html`<p class="image-version-hint">${t('image.versionLoading')}</p>`
            : html`
              <select class="form-input image-version-select"
                value=${selectedVersion}
                onChange=${(e) => setSelectedVersion(e.target.value)}
                disabled=${busy}
              >
                ${versions?.recommended && html`
                  <option value=${versions.recommended}>
                    ${versions.recommended} ★ ${t('image.versionRecommended')}
                  </option>
                `}
                ${versions?.latest && versions.latest !== versions.recommended && html`
                  <option value=${versions.latest}>
                    ${versions.latest} (${t('image.versionLatest')})
                  </option>
                `}
                ${(versions?.versions || [])
                  .filter(v => v !== versions?.recommended && v !== versions?.latest)
                  .map(v => html`<option key=${v} value=${v}>${v}</option>`)
                }
              </select>
              <button class="btn btn-primary" onClick=${handleLoadVersion} disabled=${busy}>
                ${building ? t('image.loading') : t('image.loadVersion')}
              </button>
            `
          }
        </div>
        <p style="margin:8px 0 0;color:var(--text-secondary);font-size:0.85rem">
          ${t('image.loadHint')}
        </p>
      </div>

      <div class="image-section">
        <h3 class="section-title">${t('image.currentStatus')}</h3>
        <div class="image-status-card">
          ${imageStatus?.built
            ? html`<span class="image-status-built">✅ ${t('image.built')} ${imageStatus.image}</span>`
            : html`<span class="image-status-missing">⚠️ ${t('image.notBuilt')}</span>`
          }
        </div>
      </div>

      <div class="image-actions">
        <button class="btn btn-ghost" onClick=${handleResetDefault} disabled=${busy}>
          ${pulling ? t('image.resetting') : t('image.resetDefault')}
        </button>
        <span class="image-action-hint">${t('image.resetHint')}</span>
      </div>

      ${pullLogs.length > 0 && html`
        <div class="image-section">
          <h3 class="section-title">${t('image.resetLog')}</h3>
          <div class="image-build-log">
            ${pullLogs.map((line, i) => html`<div key=${i} class="logs-line">${line}</div>`)}
          </div>
        </div>
      `}

      ${buildLogs.length > 0 && html`
        <div class="image-section">
          <h3 class="section-title">${t('image.loadLog')}</h3>
          <div class="image-build-log">
            ${buildLogs.map((line, i) => html`<div key=${i} class="logs-line">${line}</div>`)}
          </div>
        </div>
      `}
    </div>
  `;
}
