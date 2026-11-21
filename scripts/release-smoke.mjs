import https from 'node:https';
import assert from 'node:assert/strict';
import {spawn} from 'node:child_process';

const origin = process.env.API_ORIGIN || 'https://tm-smoke:8443';
const prefix = '/custom/v9';
const delay = ms => new Promise(resolve => setTimeout(resolve, ms));
function request(path, method = 'GET', data, cookie, extra = {}) {
  return new Promise((resolve, reject) => {
    const req = https.request(new URL(path, origin), {
      method, rejectUnauthorized: false, timeout: 5000,
      headers: {'Content-Type': 'application/json', ...(cookie ? {Cookie: cookie} : {}), ...extra},
    }, res => {
      const chunks = [];
      res.on('data', chunk => chunks.push(chunk));
      res.on('error', reject);
      res.on('end', () => resolve({status: res.statusCode, headers: res.headers, body: Buffer.concat(chunks)}));
    });
    req.on('error', reject);
    req.on('timeout', () => req.destroy(new Error('request timeout')));
    req.end(data ? JSON.stringify(data) : undefined);
  });
}
let ready = false;
for (let i = 0; i < 60; i++) {
  try { ready = (await request('/readyz')).status === 200; } catch {}
  if (ready) break;
  await delay(250);
}
assert(ready, 'API never became ready with the restricted runtime role');
assert.equal((await request(`${prefix}/bookings`)).status, 401);
assert.equal((await request(`${prefix}/auth/login`, 'POST', {username: 'smoke-member'})).status, 503);
assert.equal((await request(`${prefix}/auth/oauth/login`)).status, 503);
const login = await request(`${prefix}/auth/verify-otp`, 'POST', {token: 'synthetic-release-otp'});
assert.equal(login.status, 200, 'OTP login failed');
const cookie = login.headers['set-cookie']?.find(item => item.startsWith('session='))?.split(';')[0];
assert(cookie, 'login did not create a session');
const bookings = await request(`${prefix}/bookings`, 'GET', undefined, cookie);
assert.equal(bookings.status, 200);
assert.equal(JSON.parse(bookings.body).length, 100, 'collection was not bounded');
assert(JSON.parse(bookings.body).every(item => item.owner_id === 1), 'another member booking was exposed');
assert.equal(bookings.headers['x-next-offset'], '100');
const singleTrip = await request(`${prefix}/trips/1`, 'GET', undefined, cookie);
assert.equal(singleTrip.status, 200, 'single-record trip query failed');
assert.equal(JSON.parse(singleTrip.body)[0].id, 1);
assert.equal((await request(`${prefix}/refunds`, 'GET', undefined, cookie)).status, 403);
assert.equal((await request(`${prefix}/securityGroup`, 'GET', undefined, cookie)).status, 403);
const asset = await request('/main.wasm');
assert.equal(asset.status, 200);
assert(asset.body.length > 1024 && asset.headers.etag, 'WASM artifact or validator missing');
assert.equal((await request('/main.wasm', 'GET', undefined, undefined, {'If-None-Match': asset.headers.etag})).status, 304);
console.log('API readiness, OTP session, ownership, permission and artifact checks passed');

const browser = spawn('chromium', ['--headless', '--no-sandbox', '--disable-gpu', '--disable-dev-shm-usage', '--ignore-certificate-errors', '--remote-debugging-port=9222', 'about:blank'], {stdio: 'ignore'});
let socket;
try {
  let target;
  for (let i = 0; i < 60; i++) {
    try { target = await (await fetch('http://127.0.0.1:9222/json/new?about:blank', {method: 'PUT'})).json(); } catch {}
    if (target?.webSocketDebuggerUrl) break;
    await delay(100);
  }
  assert(target?.webSocketDebuggerUrl, 'headless browser unavailable');
  socket = new WebSocket(target.webSocketDebuggerUrl);
  await new Promise((resolve, reject) => {socket.addEventListener('open', resolve, {once: true}); socket.addEventListener('error', reject, {once: true});});
  let sequence = 0;
  const pending = new Map();
  const requests = [];
  const exceptions = [];
  socket.addEventListener('message', event => {
    const message = JSON.parse(event.data);
    if (message.id) {
      const resolve = pending.get(message.id);pending.delete(message.id);resolve?.(message);
    }
    if (message.method === 'Network.requestWillBeSent') requests.push(message.params.request.url);
    if (message.method === 'Runtime.exceptionThrown') exceptions.push(message.params.exceptionDetails.text);
  });
  async function send(method, params = {}) {
    const id = ++sequence;
    const result = new Promise(resolve => pending.set(id, resolve));
    socket.send(JSON.stringify({id, method, params}));
    const response = await Promise.race([result, delay(10000).then(() => {throw new Error(`browser command timed out: ${method}`);})]);
    assert(!response.error, `browser command failed: ${method}`);
    return response.result;
  }
  await send('Page.enable'); await send('Runtime.enable'); await send('Network.enable');
  await send('Network.setCookie', {name: 'session', value: cookie.slice('session='.length), url: origin, secure: true, httpOnly: true, sameSite: 'Lax'});
  await send('Page.navigate', {url: `${origin}/`});
  let rendered = false;
  for (let i = 0; i < 120; i++) {
    const result = await send('Runtime.evaluate', {expression: `!!document.querySelector('#mainContent') && document.querySelector('#userDisplay')?.textContent.includes('Smoke Member') && window.TM_CONFIG?.apiBase === '${prefix}'`, returnByValue: true});
    if (result.result?.value === true) {rendered = true;break;}
    await delay(100);
  }
  assert(rendered, 'WASM did not render the authenticated UI with the runtime API prefix');
  await send('Runtime.evaluate', {expression: "location.hash='/mybookings'"});
  const pager = "Array.from(document.querySelectorAll('.tm-pagination')).find(el=>el.getClientRects().length>0)";
  async function waitForPage(label) {
    for (let i=0;i<120;i++) {
      const result=await send('Runtime.evaluate', {expression:`(${pager})?.textContent.includes('${label}')`,returnByValue:true});
      if (result.result?.value===true) return;
      await delay(100);
    }
    assert.fail(`visible list page did not render: ${label}`);
  }
  await waitForPage('Items 1–100');
  await send('Runtime.evaluate', {expression:`Array.from((${pager}).querySelectorAll('button')).find(button=>button.textContent==='Next').click()`});
  await waitForPage('Items 101–106');
  await send('Runtime.evaluate', {expression:`Array.from((${pager}).querySelectorAll('button')).find(button=>button.textContent==='Previous').click()`});
  await waitForPage('Items 1–100');
  assert(requests.some(url => url.startsWith(`${origin}${prefix}/auth/menuUser`)), 'frontend did not use its current origin/prefix');
  assert(!requests.some(url => url.includes('localhost:8086')), 'frontend retained a compiled localhost URL');
  assert.equal(exceptions.length, 0, 'browser runtime exception');
  console.log('Real-browser WASM loading and authenticated UI checks passed');
} finally {
  socket?.close();browser.kill('SIGTERM');
}
