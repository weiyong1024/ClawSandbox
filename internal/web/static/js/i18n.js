import { useState, useEffect } from './lib.js';

const messages = {
  en: {
    'toolbar.back':          '← Dashboard',
    'toolbar.instances':     (n) => `${n} instance${n !== 1 ? 's' : ''}`,
    'toolbar.create':        '+ Create Instances',

    'dashboard.loading':     'Loading...',
    'dashboard.empty.title': 'No instances yet',
    'dashboard.empty.desc':  'Click "Create Instances" to deploy your first OpenClaw fleet.',

    'card.desktop':          '🖥 Desktop',
    'card.stop':             '⏹ Stop',
    'card.start':            '▶ Start',
    'card.destroy':          '🗑 Destroy',

    'create.title':          'Create Instances',
    'create.label':          'Number of instances',
    'create.hint':           'Each instance uses ~4 GB RAM and 2 CPU cores.',
    'create.cancel':         'Cancel',
    'create.creating':       'Creating...',
    'create.submit':         (n) => `Create ${n} Instance${n > 1 ? 's' : ''}`,

    'desktop.notFound':      'Instance not found.',
    'desktop.back':          '← Back to Dashboard',
    'desktop.newTab':        '↗ New Tab',
    'desktop.stopped':       'Instance is stopped. Start it to access the desktop.',

    'logs.title':            'Logs',
    'logs.lines':            (n) => `${n} lines`,
    'logs.waiting':          'Waiting for logs...',

    'stats.title':           'Resources',

    'toast.created':         (n) => `Created ${n} instance(s)`,
    'toast.started':         (name) => `Started ${name}`,
    'toast.stopped':         (name) => `Stopped ${name}`,
    'toast.destroyed':       (name) => `Destroyed ${name}`,
    'confirm.destroy':       (name) => `Destroy ${name}? This removes the container.`,

    'action.starting':       'Starting...',
    'action.stopping':       'Stopping...',
    'action.destroying':     'Destroying...',

    'status.disconnected':   'Connection lost. Reconnecting...',
  },

  zh: {
    'toolbar.back':          '← 仪表盘',
    'toolbar.instances':     (n) => `${n} 个实例`,
    'toolbar.create':        '+ 创建实例',

    'dashboard.loading':     '加载中...',
    'dashboard.empty.title': '暂无实例',
    'dashboard.empty.desc':  '点击「创建实例」部署你的第一个 OpenClaw 军团。',

    'card.desktop':          '🖥 桌面',
    'card.stop':             '⏹ 停止',
    'card.start':            '▶ 启动',
    'card.destroy':          '🗑 销毁',

    'create.title':          '创建实例',
    'create.label':          '实例数量',
    'create.hint':           '每个实例约占用 4 GB 内存和 2 个 CPU 核心。',
    'create.cancel':         '取消',
    'create.creating':       '创建中...',
    'create.submit':         (n) => `创建 ${n} 个实例`,

    'desktop.notFound':      '实例未找到。',
    'desktop.back':          '← 返回仪表盘',
    'desktop.newTab':        '↗ 新标签页',
    'desktop.stopped':       '实例已停止，启动后即可访问桌面。',

    'logs.title':            '日志',
    'logs.lines':            (n) => `${n} 行`,
    'logs.waiting':          '等待日志...',

    'stats.title':           '资源监控',

    'toast.created':         (n) => `已创建 ${n} 个实例`,
    'toast.started':         (name) => `已启动 ${name}`,
    'toast.stopped':         (name) => `已停止 ${name}`,
    'toast.destroyed':       (name) => `已销毁 ${name}`,
    'confirm.destroy':       (name) => `确定销毁 ${name}？这将移除容器。`,

    'action.starting':       '启动中...',
    'action.stopping':       '停止中...',
    'action.destroying':     '销毁中...',

    'status.disconnected':   '连接已断开，正在重连...',
  },
};

let currentLang = localStorage.getItem('clawsandbox-lang')
  || (navigator.language.startsWith('zh') ? 'zh' : 'en');

const listeners = new Set();

function notify() {
  for (const fn of listeners) fn(currentLang);
}

export function useLang() {
  const [lang, setLangState] = useState(currentLang);

  useEffect(() => {
    const listener = (l) => setLangState(l);
    listeners.add(listener);
    return () => listeners.delete(listener);
  }, []);

  const setLang = (l) => {
    currentLang = l;
    localStorage.setItem('clawsandbox-lang', l);
    notify();
  };

  const t = (key, ...args) => {
    const val = messages[lang]?.[key] ?? messages.en[key] ?? key;
    return typeof val === 'function' ? val(...args) : val;
  };

  return { lang, setLang, t };
}
