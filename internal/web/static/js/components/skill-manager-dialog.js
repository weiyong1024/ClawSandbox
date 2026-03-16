import { html, useState, useEffect, useCallback } from '../lib.js';
import { useLang } from '../i18n.js';
import { api } from '../api.js';

export function SkillManagerDialog({ instanceName, onClose, addToast }) {
  const { t } = useLang();
  const [skills, setSkills] = useState([]);
  const [loading, setLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState('');
  const [searchResults, setSearchResults] = useState([]);
  const [searching, setSearching] = useState(false);
  const [installing, setInstalling] = useState({});
  const [uninstalling, setUninstalling] = useState({});

  const refresh = useCallback(async () => {
    try {
      const data = await api.listInstanceSkills(instanceName);
      setSkills(data || []);
    } catch (err) {
      addToast(err.message, 'error');
    } finally {
      setLoading(false);
    }
  }, [instanceName]);

  useEffect(() => { refresh(); }, [refresh]);

  const handleSearch = async (e) => {
    e.preventDefault();
    if (!searchQuery.trim()) return;
    setSearching(true);
    try {
      const data = await api.searchClawHub(searchQuery.trim());
      setSearchResults(data || []);
    } catch (err) {
      addToast(err.message, 'error');
    } finally {
      setSearching(false);
    }
  };

  const handleInstall = async (slug) => {
    setInstalling(prev => ({ ...prev, [slug]: true }));
    try {
      await api.installSkill(instanceName, slug);
      addToast(t('skills.installed', slug), 'success');
      setSearchResults(prev => prev.filter(r => r.slug !== slug));
      refresh();
    } catch (err) {
      addToast(err.message, 'error');
    } finally {
      setInstalling(prev => { const n = { ...prev }; delete n[slug]; return n; });
    }
  };

  const handleUninstall = async (slug) => {
    setUninstalling(prev => ({ ...prev, [slug]: true }));
    try {
      await api.uninstallSkill(instanceName, slug);
      addToast(t('skills.uninstalled', slug), 'success');
      refresh();
    } catch (err) {
      addToast(err.message, 'error');
    } finally {
      setUninstalling(prev => { const n = { ...prev }; delete n[slug]; return n; });
    }
  };

  const communitySkills = skills.filter(s => !s.bundled);
  const bundledSkills = skills.filter(s => s.bundled);
  const installedSlugs = new Set(skills.map(s => s.name));

  return html`
    <div class="dialog-overlay" onClick=${onClose}>
      <div class="dialog" style="max-width:600px;max-height:85vh;display:flex;flex-direction:column" onClick=${(e) => e.stopPropagation()}>
        <div class="dialog-header">
          <h2>${t('skills.title')}: ${instanceName}</h2>
          <button class="dialog-close" onClick=${onClose}>✕</button>
        </div>
        <div class="dialog-body" style="overflow-y:auto;flex:1">
          ${loading ? html`<p>${t('dashboard.loading')}</p>` : html`
            <form onSubmit=${handleSearch} style="margin-bottom:16px">
              <div style="display:flex;gap:8px">
                <input type="text" class="form-input" style="margin-top:0;flex:1"
                  value=${searchQuery}
                  onInput=${(e) => { setSearchQuery(e.target.value); if (!e.target.value.trim()) setSearchResults([]); }}
                  placeholder=${t('skills.search')} />
                <button type="submit" class="btn btn-primary" disabled=${searching || !searchQuery.trim()}>
                  ${searching ? t('skills.searching') : t('skills.searchBtn')}
                </button>
              </div>
              <div class="form-hint" style="margin-top:4px">${t('skills.searchHint')}</div>
            </form>

            ${searchResults.length > 0 && html`
              <div style="margin-bottom:16px">
                <div class="form-label" style="margin-bottom:8px">ClawHub</div>
                <div class="skills-list">
                  ${searchResults.map(r => html`
                    <div class="skill-item" key=${r.slug}>
                      <div class="skill-info">
                        <div class="skill-name">${r.name}</div>
                        <div class="skill-meta">${r.slug}</div>
                      </div>
                      ${installedSlugs.has(r.slug) ? html`
                        <span class="skill-badge skill-badge-ready">${t('skills.ready')}</span>
                      ` : html`
                        <button class="btn btn-sm btn-success" onClick=${() => handleInstall(r.slug)}
                          disabled=${!!installing[r.slug]}>
                          ${installing[r.slug] ? t('skills.installing') : t('skills.install')}
                        </button>
                      `}
                    </div>
                  `)}
                </div>
              </div>
            `}
            ${searchResults.length === 0 && searching === false && searchQuery && html`
              <p class="form-hint">${t('skills.noResults')}</p>
            `}

            ${communitySkills.length > 0 && html`
              <div style="margin-bottom:16px">
                <div class="form-label" style="margin-bottom:8px">${t('skills.community')}</div>
                <div class="skills-list">
                  ${communitySkills.map(s => html`
                    <div class="skill-item" key=${s.name}>
                      <div class="skill-info">
                        <div class="skill-name">${s.emoji || ''} ${s.name}</div>
                        <div class="skill-meta">${s.description.length > 80 ? s.description.slice(0, 80) + '...' : s.description}</div>
                      </div>
                      <div style="display:flex;align-items:center;gap:8px">
                        <span class="skill-badge ${s.eligible ? 'skill-badge-ready' : 'skill-badge-missing'}">
                          ${s.eligible ? t('skills.ready') : t('skills.missing')}
                        </span>
                        <button class="btn btn-sm btn-danger" onClick=${() => handleUninstall(s.name)}
                          disabled=${!!uninstalling[s.name]}>
                          ${uninstalling[s.name] ? t('skills.uninstalling') : t('skills.uninstall')}
                        </button>
                      </div>
                    </div>
                  `)}
                </div>
              </div>
            `}

            <div>
              <div class="form-label" style="margin-bottom:8px">${t('skills.bundled')} (${bundledSkills.length})</div>
              <div class="skills-list">
                ${bundledSkills.map(s => html`
                  <div class="skill-item" key=${s.name}>
                    <div class="skill-info">
                      <div class="skill-name">${s.emoji || ''} ${s.name}</div>
                      <div class="skill-meta">${s.description.length > 100 ? s.description.slice(0, 100) + '...' : s.description}</div>
                    </div>
                    <span class="skill-badge ${s.eligible ? 'skill-badge-ready' : 'skill-badge-missing'}">
                      ${s.eligible ? t('skills.ready') : t('skills.missing')}
                    </span>
                  </div>
                `)}
              </div>
            </div>
          `}
        </div>
        <div class="dialog-footer">
          <button type="button" class="btn btn-ghost" onClick=${onClose}>${t('configure.cancel')}</button>
        </div>
      </div>
    </div>
  `;
}
