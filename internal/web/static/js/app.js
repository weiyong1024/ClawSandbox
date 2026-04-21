import { html, render, useState, useEffect, useCallback } from './lib.js';
import { useLang } from './i18n.js';
import { api } from './api.js';
import { connectStats, connectEvents } from './ws.js';
import { Toolbar } from './components/toolbar.js';
import { Sidebar } from './components/sidebar.js';
import { Dashboard } from './components/dashboard.js';
import { InstanceDesktop } from './components/instance-desktop.js';
import { CreateDialog } from './components/create-dialog.js';
import { ConfigureDialog } from './components/configure-dialog.js';
import { ModelAssets } from './components/model-assets.js';
import { ChannelAssets } from './components/channel-assets.js';
import { CharacterAssets } from './components/character-assets.js';
import { ImagePage } from './components/image-page.js';
import { Snapshots } from './components/snapshots.js';
import { SnapshotDialog } from './components/snapshot-dialog.js';
import { SkillManagerDialog } from './components/skill-manager-dialog.js';
import { ToastContainer, useToast } from './components/toast.js';
import { ConnectionStatus } from './components/connection-status.js';

function parseRoute(hash) {
  if (!hash || hash === '#/' || hash === '#') return { page: 'fleet', route: '#/fleet' };

  if (hash === '#/fleet/snapshots') return { page: 'snapshots', route: '#/fleet/snapshots' };

  const fleetMatch = hash.match(/^#\/fleet\/(.+)$/);
  if (fleetMatch) return { page: 'desktop', name: decodeURIComponent(fleetMatch[1]), route: '#/fleet' };

  // Also support legacy #/instance/{name}
  const legacyMatch = hash.match(/^#\/instance\/(.+)$/);
  if (legacyMatch) return { page: 'desktop', name: decodeURIComponent(legacyMatch[1]), route: '#/fleet' };

  if (hash === '#/fleet') return { page: 'fleet', route: '#/fleet' };
  if (hash === '#/assets/models') return { page: 'models', route: '#/assets/models' };
  if (hash === '#/assets/channels') return { page: 'channels', route: '#/assets/channels' };
  if (hash === '#/assets/characters') return { page: 'characters', route: '#/assets/characters' };
  if (hash === '#/system/image') return { page: 'image', route: '#/system/image' };

  return { page: 'fleet', route: '#/fleet' };
}

function App() {
  const { t } = useLang();
  const [instances, setInstances] = useState([]);
  const [stats, setStats] = useState({});
  const [view, setView] = useState(() => parseRoute(location.hash));
  const [showCreate, setShowCreate] = useState(false);
  const [loading, setLoading] = useState(true);
  const [pending, setPending] = useState({});
  const [connected, setConnected] = useState(true);
  const [configureName, setConfigureName] = useState(null);
  const [snapshotName, setSnapshotName] = useState(null);
  const [skillsName, setSkillsName] = useState(null);
  const [selected, setSelected] = useState(new Set());
  const { toasts, addToast, removeToast } = useToast();

  useEffect(() => {
    function onHash() {
      setView(parseRoute(location.hash));
    }
    window.addEventListener('hashchange', onHash);
    onHash();
    return () => window.removeEventListener('hashchange', onHash);
  }, []);

  const navigate = useCallback((target) => {
    if (target && !target.startsWith('#')) {
      // Instance name — navigate to desktop
      location.hash = `#/fleet/${encodeURIComponent(target)}`;
    } else if (target) {
      location.hash = target;
    } else {
      location.hash = '#/fleet';
    }
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

  const onCreate = async (count, snapshotName, runtime) => {
    try {
      await api.createInstances(count, snapshotName, runtime);
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
        if (view.page === 'desktop' && view.name === name) navigate('#/fleet');
      } catch (err) {
        addToast(err.message, 'error');
      }
    })();
  };

  const onConsole = (name) => {
    const inst = instances.find(i => i.name === name);
    if (inst && inst.gateway_port) {
      window.open(`http://localhost:${inst.gateway_port}/`, '_blank');
    } else {
      window.open(`/console/${name}/`, '_blank');
    }
  };

  const onHermesDashboard = (name) => {
    const inst = instances.find(i => i.name === name);
    if (inst && inst.hermes_dashboard_port) {
      window.open(`http://localhost:${inst.hermes_dashboard_port}/`, '_blank');
    }
  };

  const onRestartBot = async (name) => {
    if (!confirm(t('confirm.restartBot', name))) return;
    await withPending(name, 'restarting', async () => {
      try {
        await api.restartBot(name);
        addToast(t('toast.restartedBot', name), 'success');
      } catch (err) {
        addToast(err.message, 'error');
      }
    })();
  };

  const onSnapshot = (name) => {
    setSnapshotName(name);
  };

  const onConfigure = async (name, config) => {
    try {
      await api.configureInstance(name, config);
      addToast(t('configure.success', name), 'success');
      setConfigureName(null);
      refresh();
    } catch (err) {
      addToast(err.message, 'error');
    }
  };

  const onToggleSelect = useCallback((name) => {
    setSelected(prev => {
      const next = new Set(prev);
      if (next.has(name)) next.delete(name); else next.add(name);
      return next;
    });
  }, []);

  const onSelectAll = useCallback(() => {
    setSelected(prev => {
      if (prev.size === instances.length) return new Set();
      return new Set(instances.map(i => i.name));
    });
  }, [instances]);

  const onBatchDestroy = async () => {
    const names = [...selected];
    if (names.length === 0) return;
    if (!confirm(t('confirm.batchDestroy', names.length))) return;
    for (const name of names) {
      setPending(p => ({ ...p, [name]: 'destroying' }));
    }
    try {
      const result = await api.batchDestroyInstances(names);
      addToast(t('toast.batchDestroyed', result.destroyed), 'success');
    } catch (err) {
      addToast(err.message, 'error');
    }
    setPending({});
    setSelected(new Set());
  };

  useEffect(() => {
    const onKey = (e) => {
      if (e.key === 'Escape' && showCreate) setShowCreate(false);
      if (e.key === 'Escape' && configureName) setConfigureName(null);
      if (e.key === 'Escape' && snapshotName) setSnapshotName(null);
      if (e.key === 'Escape' && skillsName) setSkillsName(null);
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [showCreate, configureName, snapshotName, skillsName]);

  const currentInstance = view.page === 'desktop'
    ? instances.find(i => i.name === view.name)
    : null;

  // Desktop view has its own full-screen layout (no sidebar)
  if (view.page === 'desktop') {
    return html`
      <${Toolbar} />
      <${InstanceDesktop}
        instance=${currentInstance}
        stats=${stats[view.name]}
        pending=${pending[view.name]}
        onStart=${onStart}
        onStop=${onStop}
        onBack=${() => navigate('#/fleet')}
      />
      <${ToastContainer} toasts=${toasts} onDismiss=${removeToast} />
      <${ConnectionStatus} connected=${connected} />
    `;
  }

  // Standard sidebar layout
  let content;
  switch (view.page) {
    case 'models':
      content = html`<${ModelAssets} addToast=${addToast} />`;
      break;
    case 'channels':
      content = html`<${ChannelAssets} addToast=${addToast} />`;
      break;
    case 'characters':
      content = html`<${CharacterAssets} addToast=${addToast} />`;
      break;
    case 'image':
      content = html`<${ImagePage} addToast=${addToast} />`;
      break;
    case 'snapshots':
      content = html`<${Snapshots} addToast=${addToast} />`;
      break;
    case 'fleet':
    default:
      content = html`
        <${Dashboard}
          instances=${instances}
          stats=${stats}
          loading=${loading}
          pending=${pending}
          selected=${selected}
          onToggleSelect=${onToggleSelect}
          onSelectAll=${onSelectAll}
          onBatchDestroy=${onBatchDestroy}
          onStart=${onStart}
          onStop=${onStop}
          onDestroy=${onDestroy}
          onDesktop=${navigate}
          onConsole=${onConsole}
          onRestartBot=${onRestartBot}
          onConfigure=${(name) => setConfigureName(name)}
          onSnapshot=${onSnapshot}
          onSkills=${(name) => setSkillsName(name)}
          onHermesDashboard=${onHermesDashboard}
          onCreateClick=${() => setShowCreate(true)}
        />
      `;
      break;
  }

  return html`
    <${Toolbar} />
    <div class="app-layout">
      <${Sidebar} currentRoute=${view.route} onNavigate=${navigate} />
      <main class="app-main">
        ${content}
      </main>
    </div>
    ${showCreate && html`
      <${CreateDialog} onClose=${() => setShowCreate(false)} onCreate=${onCreate} />
    `}
    ${configureName && html`
      <${ConfigureDialog}
        instanceName=${configureName}
        currentModelAssetId=${(instances.find(i => i.name === configureName) || {}).model_asset_id || ''}
        currentCharacterAssetId=${(instances.find(i => i.name === configureName) || {}).character_asset_id || ''}
        onClose=${() => setConfigureName(null)}
        onConfigure=${onConfigure}
      />
    `}
    ${skillsName && html`
      <${SkillManagerDialog}
        instanceName=${skillsName}
        onClose=${() => setSkillsName(null)}
        addToast=${addToast}
      />
    `}
    ${snapshotName && html`
      <${SnapshotDialog}
        instanceName=${snapshotName}
        onClose=${() => setSnapshotName(null)}
        addToast=${addToast}
      />
    `}
    <${ToastContainer} toasts=${toasts} onDismiss=${removeToast} />
    <${ConnectionStatus} connected=${connected} />
  `;
}

render(html`<${App} />`, document.getElementById('app'));
