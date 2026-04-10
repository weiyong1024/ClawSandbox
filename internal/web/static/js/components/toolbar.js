import { html, useState, useRef, useEffect } from '../lib.js';
import { useLang } from '../i18n.js';
import { api } from '../api.js';

const LANGUAGES = [
  { code: 'en', label: 'English' },
  { code: 'zh', label: '简体中文' },
];

export function Toolbar() {
  const { lang, setLang } = useLang();
  const [open, setOpen] = useState(false);
  const [version, setVersion] = useState('');
  const ref = useRef(null);

  useEffect(() => {
    api.version().then(d => setVersion(d.version || '')).catch(() => {});
  }, []);

  useEffect(() => {
    if (!open) return;
    const onClickOutside = (e) => {
      if (ref.current && !ref.current.contains(e.target)) setOpen(false);
    };
    document.addEventListener('mousedown', onClickOutside);
    return () => document.removeEventListener('mousedown', onClickOutside);
  }, [open]);

  const current = LANGUAGES.find(l => l.code === lang) || LANGUAGES[0];

  return html`
    <header class="toolbar">
      <div class="toolbar-brand">
        <span class="toolbar-logo">🦞</span>
        <h1 class="toolbar-title">ClawFleet</h1>
        ${version && html`<span class="toolbar-version">${version}</span>`}
      </div>
      <div class="toolbar-right">
        <div class="lang-dropdown" ref=${ref}>
          <button class="btn btn-ghost btn-sm lang-trigger" onClick=${() => setOpen(!open)}>
            🌐 ${current.label}
          </button>
          ${open && html`
            <div class="lang-menu">
              ${LANGUAGES.map(l => html`
                <button
                  key=${l.code}
                  class="lang-option ${l.code === lang ? 'lang-option-active' : ''}"
                  onClick=${() => { setLang(l.code); setOpen(false); }}
                >${l.label}</button>
              `)}
            </div>
          `}
        </div>
      </div>
    </header>
  `;
}
