const BASE = '/api/v1';

async function request(method, path, body) {
  const opts = { method, headers: {} };
  if (body !== undefined) {
    opts.headers['Content-Type'] = 'application/json';
    opts.body = JSON.stringify(body);
  }
  const res = await fetch(BASE + path, opts);
  const json = await res.json();
  if (!res.ok) throw new Error(json.error?.message || res.statusText);
  return json.data;
}

export const api = {
  listInstances:  ()            => request('GET',    '/instances'),
  createInstances:(count)       => request('POST',   '/instances', { count }),
  startInstance:  (name)        => request('POST',   `/instances/${name}/start`),
  stopInstance:   (name)        => request('POST',   `/instances/${name}/stop`),
  destroyInstance:(name, purge) => request('DELETE',  `/instances/${name}${purge ? '?purge=true' : ''}`),
  imageStatus:    ()            => request('GET',    '/image/status'),
};
