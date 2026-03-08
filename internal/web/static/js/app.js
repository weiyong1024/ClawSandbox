import { html, render, useState, useEffect, useCallback } from './lib.js';
import { useLang } from './i18n.js';
import { api } from './api.js';
import { connectStats, connectEvents } from './ws.js';
import { Toolbar } from './components/toolbar.js';
import { Dashboard } from './components/dashboard.js';
import { InstanceDesktop } from './components/instance-desktop.js';
import { CreateDialog } from './components/create-dialog.js';
import { ToastContainer, useToast } from './components/toast.js';
import { ConnectionStatus } from './components/connection-status.js';

function App() {
  const { t } = useLang();
  const [instances, setInstances] = useState([]);
  const [stats, setStats] = useState({});
  const [view, setView] = useState({ page: 'dashboard' });
  const [showCreate, setShowCreate] = useState(false);
  const [loading, setLoading] = useState(true);
  const [pending, setPending] = useState({});
  const [connected, setConnected] = useState(true);
  const { toasts, addToast, removeToast } = useToast();

  useEffect(() => {
    function onHash() {
      const match = location.hash.match(/^#\/instance\/(.+)$/);
      setView(match ? { page: 'desktop', name: decodeURIComponent(match[1]) } : { page: 'dashboard' });
    }
    window.addEventListener('hashchange', onHash);
    onHash();
    return () => window.removeEventListener('hashchange', onHash);
  }, []);

  const navigate = useCallback((name) => {
    location.hash = name ? `#/instance/${encodeURIComponent(name)}` : '#/';
  }, []);

  const refresh = useCallback(async () => {
    try {
      const data = await api.listInstances();
      setInstances(data);
    } catch (err) {
      addToast(err.message, 'error');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    refresh();
    const statsConn = connectStats(
      (msg) => {
        const map = {};
        for (const s of msg.instances) map[s.name] = s;
        setStats(map);
      },
      (status) => setConnected(status),
    );
    const eventsConn = connectEvents(() => refresh());
    return () => { statsConn.close(); eventsConn.close(); };
  }, [refresh]);

  const withPending = (name, action, fn) => async () => {
    setPending(p => ({ ...p, [name]: action }));
    try { await fn(); } finally { setPending(p => { const n = { ...p }; delete n[name]; return n; }); }
  };

  const onCreate = async (count) => {
    try {
      await api.createInstances(count);
      addToast(t('toast.created', count), 'success');
      setShowCreate(false);
    } catch (err) {
      addToast(err.message, 'error');
    }
  };

  const onStart = (name) => withPending(name, 'starting', async () => {
    try {
      await api.startInstance(name);
      addToast(t('toast.started', name), 'success');
    } catch (err) {
      addToast(err.message, 'error');
    }
  })();

  const onStop = (name) => withPending(name, 'stopping', async () => {
    try {
      await api.stopInstance(name);
      addToast(t('toast.stopped', name), 'success');
    } catch (err) {
      addToast(err.message, 'error');
    }
  })();

  const onDestroy = async (name) => {
    if (!confirm(t('confirm.destroy', name))) return;
    await withPending(name, 'destroying', async () => {
      try {
        await api.destroyInstance(name);
        addToast(t('toast.destroyed', name), 'success');
        if (view.page === 'desktop' && view.name === name) navigate(null);
      } catch (err) {
        addToast(err.message, 'error');
      }
    })();
  };

  useEffect(() => {
    const onKey = (e) => {
      if (e.key === 'Escape' && showCreate) setShowCreate(false);
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [showCreate]);

  const currentInstance = view.page === 'desktop'
    ? instances.find(i => i.name === view.name)
    : null;

  return html`
    <${Toolbar}
      count=${instances.length}
      onCreateClick=${() => setShowCreate(true)}
      showBack=${view.page === 'desktop'}
      onBack=${() => navigate(null)}
    />
    ${view.page === 'dashboard' ? html`
      <${Dashboard}
        instances=${instances}
        stats=${stats}
        loading=${loading}
        pending=${pending}
        onStart=${onStart}
        onStop=${onStop}
        onDestroy=${onDestroy}
        onDesktop=${navigate}
      />
    ` : html`
      <${InstanceDesktop}
        instance=${currentInstance}
        stats=${stats[view.name]}
        pending=${pending[view.name]}
        onStart=${onStart}
        onStop=${onStop}
        onBack=${() => navigate(null)}
      />
    `}
    ${showCreate && html`
      <${CreateDialog} onClose=${() => setShowCreate(false)} onCreate=${onCreate} />
    `}
    <${ToastContainer} toasts=${toasts} onDismiss=${removeToast} />
    <${ConnectionStatus} connected=${connected} />
  `;
}

render(html`<${App} />`, document.getElementById('app'));
