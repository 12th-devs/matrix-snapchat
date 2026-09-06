// Inspect Beeper's decoded media through its supported local OAuth API.
// Credentials stay in memory; this never reads bridge credentials or opens snaps.
import { createServer } from 'node:http';
import { randomBytes, createHash } from 'node:crypto';
import { execFile } from 'node:child_process';

const args = process.argv.slice(2);
const focus = args.includes('--focus');
const pairs = [];
for (let index = 0; index + 1 < args.length; index += 2) {
  if (args[index] === '--focus') break;
  pairs.push({ chatID: args[index], messageID: args[index + 1] });
}
if (pairs.length === 0) {
  console.error('Usage: node scripts/check_beeper_media.mjs <Matrix room ID> <Matrix event ID> [...] [--focus]');
  process.exit(1);
}
const base = 'http://127.0.0.1:23373';
const verifier = randomBytes(32).toString('base64url');
const state = randomBytes(24).toString('hex');
let resolveCode;
const codePromise = new Promise(resolve => { resolveCode = resolve; });
const server = createServer((req, res) => {
  const url = new URL(req.url, 'http://127.0.0.1');
  if (url.pathname !== '/callback' || url.searchParams.get('state') !== state) {
    res.writeHead(400).end('Invalid OAuth callback');
    return;
  }
  res.setHeader('Content-Type', 'text/plain');
  res.end('Beeper media check authorized. You can close this page.');
  resolveCode(url.searchParams.get('code'));
});
await new Promise(resolve => server.listen(0, '127.0.0.1', resolve));
const redirect = `http://127.0.0.1:${server.address().port}/callback`;
async function request(path, options = {}) {
  const response = await fetch(base + path, { ...options, signal: AbortSignal.timeout(30000) });
  if (!response.ok) throw new Error(`${path.split('?')[0]} returned HTTP ${response.status}`);
  return response.status === 204 ? null : response.json();
}
let timer;
try {
  const client = await request('/oauth/register', {
    method: 'POST', headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ client_name: 'Snapchat Bridge Media Check', redirect_uris: [redirect], scope: focus ? 'read write' : 'read' }),
  });
  const query = new URLSearchParams({ client_id: client.client_id, redirect_uri: redirect,
    response_type: 'code', scope: focus ? 'read write' : 'read', state,
    code_challenge: createHash('sha256').update(verifier).digest('base64url'), code_challenge_method: 'S256' });
  const authURL = `${base}/oauth/authorize?${query}`;
  console.log('Approve "Snapchat Bridge Media Check" in Beeper. Waiting up to 3 minutes.');
  if (process.platform === 'win32') {
    execFile('powershell.exe', ['-NoProfile', '-Command', `Start-Process '${authURL}'`]);
  } else {
    console.log(authURL);
  }
  const code = await Promise.race([codePromise, new Promise((_, reject) => {
    timer = setTimeout(() => reject(new Error('OAuth approval timed out')), 180000);
  })]);
  if (!code) throw new Error('OAuth authorization was declined');
  const token = await request('/oauth/token', {
    method: 'POST', headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body: new URLSearchParams({ grant_type: 'authorization_code', client_id: client.client_id,
      code, code_verifier: verifier, redirect_uri: redirect }),
  });
  const headers = { Authorization: `Bearer ${token.access_token}` };
  for (const pair of pairs) {
    const message = await request(`/v1/chats/${encodeURIComponent(pair.chatID)}/messages/${encodeURIComponent(pair.messageID)}`, { headers });
    console.log(JSON.stringify({ room: pair.chatID, event: pair.messageID, decoded: message }, (key, value) => {
      if (/token|key|iv|hashes/i.test(key)) return undefined;
      if (typeof value === 'string' && value.includes('?encryptedFileInfoJSON=')) return value.split('?')[0];
      return value;
    }, 2));
  }
  if (focus) {
    const pair = pairs[pairs.length - 1];
    await request('/v1/focus', { method: 'POST', headers: { ...headers, 'Content-Type': 'application/json' },
      body: JSON.stringify({ chatID: pair.chatID, messageID: pair.messageID }) });
    console.log('Focused the message in Beeper. Confirm the image is actually visible.');
  }
} catch (error) {
  console.error(error.message);
  process.exitCode = 1;
} finally {
  clearTimeout(timer);
  server.closeAllConnections();
  server.close();
}
