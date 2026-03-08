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
  listInstances:  ()            => request('GET',    '/instances'),
  createInstances:(count)       => request('POST',   '/instances', { count }),
  startInstance:  (name)        => request('POST',   `/instances/${encodeURIComponent(name)}/start`),
  stopInstance:   (name)        => request('POST',   `/instances/${encodeURIComponent(name)}/stop`),
  destroyInstance:(name, purge) => request('DELETE',  `/instances/${encodeURIComponent(name)}${purge ? '?purge=true' : ''}`),
  imageStatus:    ()            => request('GET',    '/image/status'),
};
