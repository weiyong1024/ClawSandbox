import { useState, useEffect } from './lib.js';

const messages = {
  en: {
    'toolbar.back':          '← Dashboard',
    'toolbar.instances':     (n) => `${n} instance${n !== 1 ? 's' : ''}`,
    'toolbar.create':        '+ Create Instances',

    // Sidebar
    'sidebar.assets':        'Assets',
    'sidebar.models':        'Model Config',
    'sidebar.channels':      'Channel Config',
    'sidebar.fleet':         'Fleet',
    'sidebar.instances':     'Instances',
    'sidebar.system':        'System',
    'sidebar.image':         'Image',

    'dashboard.loading':     'Loading...',
    'dashboard.empty.title': 'No instances yet',
    'dashboard.empty.desc':  'Click "Create Instances" to deploy your first OpenClaw fleet.',

    'card.desktop':          '🖥 Desktop',
    'card.stop':             '⏹ Stop',
    'card.start':            '▶ Start',
    'card.destroy':          '🗑 Destroy',
    'card.unconfigured':     'Not configured',

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
    'action.configuring':    'Configuring...',

    'card.configure':        'Configure',

    'configure.title':       'Configure Instance',
    'configure.provider':    'Provider',
    'configure.apiKey':      'API Key',
    'configure.apiKeyHint':  'Required to reconfigure; not shown for security',
    'configure.model':       'Model',
    'configure.modelHint':   'e.g. gpt-4o, claude-sonnet-4-6',
    'configure.channel':     'Channel',
    'configure.channelToken':'Channel Bot Token',
    'configure.channelTokenHint': 'Required to reconfigure; not shown for security',
    'configure.cancel':      'Cancel',
    'configure.submit':      'Configure',
    'configure.configuring': 'Configuring...',
    'configure.success':     (name) => `${name} configured successfully`,
    'configure.modelConfig': 'Model Config',
    'configure.channelConfig': 'Channel Config (optional)',
    'configure.noModels':    'No model configs available. Add one in Assets → Model Config.',
    'configure.noChannel':   'None',
    'configure.unavailableModels': (n) => `${n} more config(s) used by other instances`,

    // Assets
    'assets.addModel':       '+ Add Model Config',
    'assets.editModel':      'Edit Model Config',
    'assets.addChannel':     '+ Add Channel Config',
    'assets.editChannel':    'Edit Channel Config',
    'assets.noModels':       'No model configs yet',
    'assets.noModelsDesc':   'Add your LLM API key and model to get started.',
    'assets.noChannels':     'No channel configs yet',
    'assets.noChannelsDesc': 'Add a bot token to connect your instances to messaging platforms.',
    'assets.name':           'Name',
    'assets.nameHint':       'e.g. GPT-4o Production',
    'assets.channelNameHint':'e.g. TG Bot 1',
    'assets.customModel':    'Custom...',
    'assets.customModelName':'Custom Model Name',
    'assets.test':           'Test',
    'assets.testing':        'Testing...',
    'assets.testSuccess':    'Validation passed',
    'assets.testFailed':     'Validation failed',
    'assets.mustValidate':   'Please test and validate before saving.',
    'assets.edit':           'Edit',
    'assets.delete':         'Delete',
    'assets.saving':         'Saving...',
    'assets.created':        'Config created',
    'assets.updated':        'Config updated',
    'assets.deleted':        (name) => `${name} deleted`,
    'assets.confirmDelete':  (name) => `Delete "${name}"?`,
    'assets.status':         'Status',
    'assets.validated':      'Validated',
    'assets.unvalidated':    'Not validated',
    'assets.usedBy':         'Used by',

    // Image
    'image.selectFlavor':    'Select Image Flavor',
    'image.recommended':     'Recommended',
    'image.baseImage':       'Base',
    'image.size':            'Size',
    'image.desktop':         'Desktop',
    'image.currentStatus':   'Current Status',
    'image.built':           'Built',
    'image.notBuilt':        'Image not built yet',
    'image.build':           'Build Image',
    'image.building':        'Building...',
    'image.buildLog':        'Build Log',
    'image.buildSuccess':    'Image built successfully',
    'image.buildFailed':     'Image build failed',

    'status.disconnected':   'Connection lost. Reconnecting...',
  },

  zh: {
    'toolbar.back':          '← 仪表盘',
    'toolbar.instances':     (n) => `${n} 个实例`,
    'toolbar.create':        '+ 创建实例',

    // Sidebar
    'sidebar.assets':        '资产管理',
    'sidebar.models':        'Model 配置',
    'sidebar.channels':      'Channel 配置',
    'sidebar.fleet':         '实例管理',
    'sidebar.instances':     '实例列表',
    'sidebar.system':        '系统',
    'sidebar.image':         '镜像管理',

    'dashboard.loading':     '加载中...',
    'dashboard.empty.title': '暂无实例',
    'dashboard.empty.desc':  '点击「创建实例」部署你的第一个 OpenClaw 军团。',

    'card.desktop':          '🖥 桌面',
    'card.stop':             '⏹ 停止',
    'card.start':            '▶ 启动',
    'card.destroy':          '🗑 销毁',
    'card.unconfigured':     '未配置',

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
    'action.configuring':    '配置中...',

    'card.configure':        '配置',

    'configure.title':       '配置实例',
    'configure.provider':    '提供商',
    'configure.apiKey':      'API 密钥',
    'configure.apiKeyHint':  '重新配置时需填写，出于安全不回显',
    'configure.model':       '模型',
    'configure.modelHint':   '例如 gpt-4o, claude-sonnet-4-6',
    'configure.channel':     '频道',
    'configure.channelToken':'频道机器人令牌',
    'configure.channelTokenHint': '重新配置时需填写，出于安全不回显',
    'configure.cancel':      '取消',
    'configure.submit':      '配置',
    'configure.configuring': '配置中...',
    'configure.success':     (name) => `${name} 配置成功`,
    'configure.modelConfig': 'Model 配置',
    'configure.channelConfig': 'Channel 配置（可选）',
    'configure.noModels':    '暂无可用的 Model 配置。请先在「资产管理 → Model 配置」中添加。',
    'configure.noChannel':   '无',
    'configure.unavailableModels': (n) => `另有 ${n} 个配置已被其他实例使用`,

    // Assets
    'assets.addModel':       '+ 添加 Model 配置',
    'assets.editModel':      '编辑 Model 配置',
    'assets.addChannel':     '+ 添加 Channel 配置',
    'assets.editChannel':    '编辑 Channel 配置',
    'assets.noModels':       '暂无 Model 配置',
    'assets.noModelsDesc':   '添加你的 LLM API Key 和模型以开始使用。',
    'assets.noChannels':     '暂无 Channel 配置',
    'assets.noChannelsDesc': '添加机器人令牌以连接消息平台。',
    'assets.name':           '名称',
    'assets.nameHint':       '例如 GPT-4o 生产',
    'assets.channelNameHint':'例如 TG Bot 1',
    'assets.customModel':    '自定义...',
    'assets.customModelName':'自定义模型名称',
    'assets.test':           '测试',
    'assets.testing':        '测试中...',
    'assets.testSuccess':    '验证通过',
    'assets.testFailed':     '验证失败',
    'assets.mustValidate':   '请先测试验证后再保存。',
    'assets.edit':           '编辑',
    'assets.delete':         '删除',
    'assets.saving':         '保存中...',
    'assets.created':        '配置已创建',
    'assets.updated':        '配置已更新',
    'assets.deleted':        (name) => `${name} 已删除`,
    'assets.confirmDelete':  (name) => `确定删除「${name}」？`,
    'assets.status':         '状态',
    'assets.validated':      '已验证',
    'assets.unvalidated':    '未验证',
    'assets.usedBy':         '使用者',

    // Image
    'image.selectFlavor':    '选择镜像方案',
    'image.recommended':     '推荐',
    'image.baseImage':       '基础镜像',
    'image.size':            '大小',
    'image.desktop':         '桌面',
    'image.currentStatus':   '当前状态',
    'image.built':           '已构建',
    'image.notBuilt':        '镜像尚未构建',
    'image.build':           '构建镜像',
    'image.building':        '构建中...',
    'image.buildLog':        '构建日志',
    'image.buildSuccess':    '镜像构建成功',
    'image.buildFailed':     '镜像构建失败',

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
