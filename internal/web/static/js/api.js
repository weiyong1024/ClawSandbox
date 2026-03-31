const BASE = '/api/v1';

async function request(method, path, body) {
  const opts = { method, headers: {} };
  if (body !== undefined) {
    opts.headers['Content-Type'] = 'application/json';
    opts.body = JSON.stringify(body);
  }
  const res = await fetch(BASE + path, opts);
  let json;
  try {
    json = await res.json();
  } catch {
    throw new Error(res.statusText || `HTTP ${res.status}`);
  }
  if (!res.ok) throw new Error(json.error?.message || res.statusText);
  return json.data;
}

export const api = {
  // Instances
  listInstances:  ()            => request('GET',    '/instances'),
  createInstances:(count, snapshotName) => request('POST', '/instances', { count, ...(snapshotName && { snapshot_name: snapshotName }) }),
  startInstance:  (name)        => request('POST',   `/instances/${encodeURIComponent(name)}/start`),
  stopInstance:   (name)        => request('POST',   `/instances/${encodeURIComponent(name)}/stop`),
  destroyInstance:(name)        => request('DELETE',  `/instances/${encodeURIComponent(name)}`),
  batchDestroyInstances:(names) => request('POST',  '/instances/batch-destroy', { names }),
  resetInstance:  (name)        => request('POST',   `/instances/${encodeURIComponent(name)}/reset`),
  restartBot:     (name)        => request('POST',   `/instances/${encodeURIComponent(name)}/restart-bot`),
  configureInstance: (name, config) => request('POST', `/instances/${encodeURIComponent(name)}/configure`, config),
  getConfigStatus:   (name)        => request('GET',  `/instances/${encodeURIComponent(name)}/configure/status`),

  // Image
  imageStatus:      () => request('GET', '/image/status'),
  openclawVersions: () => request('GET', '/image/openclaw-versions'),

  // Model assets
  listModelAssets:  ()           => request('GET',    '/assets/models'),
  createModelAsset: (data)       => request('POST',   '/assets/models', data),
  updateModelAsset: (id, data)   => request('PUT',    `/assets/models/${encodeURIComponent(id)}`, data),
  deleteModelAsset: (id)         => request('DELETE', `/assets/models/${encodeURIComponent(id)}`),
  testModelAsset:   (data)       => request('POST',   '/assets/models/test', data),
  startCodexOAuth:  (model, name) => request('POST', '/oauth/codex/start', { model, name }),
  pollCodexOAuth:   (state) => request('GET',  `/oauth/codex/poll?state=${encodeURIComponent(state)}`),

  // Channel assets
  listChannelAssets:  ()           => request('GET',    '/assets/channels'),
  createChannelAsset: (data)       => request('POST',   '/assets/channels', data),
  updateChannelAsset: (id, data)   => request('PUT',    `/assets/channels/${encodeURIComponent(id)}`, data),
  deleteChannelAsset: (id)         => request('DELETE', `/assets/channels/${encodeURIComponent(id)}`),
  testChannelAsset:   (data)       => request('POST',   '/assets/channels/test', data),

  // Skills (per-instance)
  listInstanceSkills: (name)       => request('GET',    `/instances/${encodeURIComponent(name)}/skills`),
  installSkill:       (name, slug) => request('POST',   `/instances/${encodeURIComponent(name)}/skills/install`, { slug }),
  uninstallSkill:     (name, slug) => request('DELETE', `/instances/${encodeURIComponent(name)}/skills/${encodeURIComponent(slug)}`),
  searchClawHub:      (q)          => request('GET',    `/skills/search?q=${encodeURIComponent(q)}`),

  // Character assets
  listCharacterAssets:  ()           => request('GET',    '/assets/characters'),
  createCharacterAsset: (data)       => request('POST',   '/assets/characters', data),
  updateCharacterAsset: (id, data)   => request('PUT',    `/assets/characters/${encodeURIComponent(id)}`, data),
  deleteCharacterAsset: (id)         => request('DELETE', `/assets/characters/${encodeURIComponent(id)}`),

  // Snapshots
  listSnapshots:  ()     => request('GET',    '/snapshots'),
  createSnapshot: (data) => request('POST',   '/snapshots', data),
  deleteSnapshot: (id)   => request('DELETE', `/snapshots/${encodeURIComponent(id)}`),
};
