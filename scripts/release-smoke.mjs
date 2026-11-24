const origin = Deno.env.get('API_ORIGIN') || 'https://tm-smoke:8443';
const prefix = '/custom/v9';
const delay = ms => new Promise(resolve => setTimeout(resolve, ms));

function assert(condition, message = 'assertion failed') {
  if (!condition) throw new Error(message);
}
assert.equal = (actual, expected, message = 'unexpected value') =>
  assert(Object.is(actual, expected), `${message}: got ${JSON.stringify(actual)}, expected ${JSON.stringify(expected)}`);
assert.deepEqual = (actual, expected, message = 'unexpected JSON') =>
  assert.equal(JSON.stringify(actual), JSON.stringify(expected), message);
assert.match = (value, pattern, message) => assert(pattern.test(value), message);
assert.fail = message => { throw new Error(message); };

async function request(path, method = 'GET', data, cookie, extra = {}) {
  const res = await fetch(new URL(path, origin), {
    method, redirect: 'manual', signal: AbortSignal.timeout(5000),
    headers: {'Content-Type': 'application/json', ...(cookie ? {Cookie: cookie} : {}), ...extra},
    body: data ? JSON.stringify(data) : undefined,
  });
  const headers = Object.fromEntries(res.headers);
  headers['set-cookie'] = res.headers.getSetCookie();
  const body = path === '/main.wasm' ? await res.bytes() : await res.text();
  return {status: res.status, headers, body};
}
let ready = false;
let readinessError;
for (let i = 0; i < 60; i++) {
  try {
    const {status} = await request('/readyz');
    ready = status === 200;
    readinessError = 'HTTP ' + status;
  } catch (error) { readinessError = error.message; }
  if (ready) break;
  await delay(250);
}
assert(ready, 'API never became ready with the restricted runtime role: ' + readinessError);
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
const lookup = await request(`${prefix}/bookings/1/bookingPeople/lookup?q=Other%20Member`, 'GET', undefined, cookie);
assert.equal(lookup.status, 200);
assert.deepEqual(JSON.parse(lookup.body), [{id: 2, name: 'Other Member', username: ''}]);
const otherLookup = await request(`${prefix}/bookings/2/bookingPeople/lookup?q=Other%20Member`, 'GET', undefined, cookie);
assert.equal(otherLookup.status, 200);
assert.deepEqual(JSON.parse(otherLookup.body), [], 'another member booking opened a participant lookup');
const singleTrip = await request(`${prefix}/trips/1`, 'GET', undefined, cookie);
const tripList = await request(`${prefix}/trips?pagination=cursor&after=0&limit=100`, 'GET', undefined, cookie);
assert.equal(tripList.status, 200, `trip list failed: ${tripList.body}`);
assert.equal(JSON.parse(tripList.body)[0].trip_name, 'Synthetic trip');
const tripReport = await request(`${prefix}/tripsReport?limit=1`, 'GET', undefined, cookie);
assert.equal(tripReport.status, 200, `Trip List report failed: ${tripReport.body}`);
assert.equal(JSON.parse(tripReport.body)[0].trip_id, 1);
assert.equal(tripReport.headers['x-next-offset'], '1');
const tripReportNext = await request(`${prefix}/tripsReport?limit=1&offset=1`, 'GET', undefined, cookie);
assert.equal(tripReportNext.status, 200);
assert.equal(JSON.parse(tripReportNext.body)[0].person_name, 'Other Member');
assert.equal(singleTrip.status, 200, 'single-record trip query failed');
assert.equal(JSON.parse(singleTrip.body)[0].id, 1);
assert.equal((await request(`${prefix}/refunds`, 'GET', undefined, cookie)).status, 403);
assert.equal((await request(`${prefix}/securityGroup`, 'GET', undefined, cookie)).status, 403);
assert.equal((await request(`${prefix}/operations`, 'GET', undefined, cookie)).status, 403, 'member accessed operational metrics');
assert.match(bookings.headers['x-request-id'] || '', /^[a-f0-9]{24}$/, 'request correlation header missing');
const asset = await request('/main.wasm');
assert.equal(asset.status, 200);
assert(asset.body.length > 1024 && asset.headers.etag, 'WASM artifact or validator missing');
assert.equal((await request('/main.wasm', 'GET', undefined, undefined, {'If-None-Match': asset.headers.etag})).status, 304);
console.log('API readiness, OTP session, ownership, permission and artifact checks passed');

const browser = new Deno.Command('/usr/local/bin/smoke-chromium', {args: ['--headless', '--no-sandbox', '--disable-gpu', '--disable-dev-shm-usage', '--ignore-certificate-errors', '--remote-debugging-port=9222', '--user-data-dir=/tmp/tm-chromium', 'about:blank'], clearEnv: true, env: {HOME: '/tmp', PATH: '/usr/bin:/bin'}, stdout: 'null', stderr: 'null'}).spawn();
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
  const browserConsole=[];
  socket.addEventListener('message', event => {
    const message = JSON.parse(event.data);
    if (message.id) {
      const resolve = pending.get(message.id);pending.delete(message.id);resolve?.(message);
    }
    if (message.method === 'Network.requestWillBeSent') requests.push(message.params.request.url);
    if (message.method === 'Runtime.exceptionThrown') exceptions.push(message.params.exceptionDetails.exception?.description||message.params.exceptionDetails.text);
    if (message.method === 'Runtime.consoleAPICalled') browserConsole.push(message.params.args.map(a=>a.value??a.description??a.type).join(' '));
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
  await send('Page.navigate', {url: `${origin}/?month=2026-01`});
  let rendered = false;
  for (let i = 0; i < 120; i++) {
    const result = await send('Runtime.evaluate', {expression: `!!document.querySelector('#mainContent') && document.querySelector('#userDisplay')?.textContent.includes('Smoke Member') && window.TM_CONFIG?.apiBase === '${prefix}'`, returnByValue: true});
    if (result.result?.value === true) {rendered = true;break;}
    await delay(100);
  }
  if(!rendered){const diagnostic=await send('Runtime.evaluate',{expression:"JSON.stringify({text:document.body.textContent.slice(0,1400),user:document.querySelector('#userDisplay')?.textContent,base:window.TM_CONFIG?.apiBase})",returnByValue:true});assert.fail('WASM did not render the authenticated UI: '+diagnostic.result?.value+' exceptions='+exceptions.join(' | ')+' console='+browserConsole.slice(-12).join(' | '));}
  await waitUI("document.querySelector('#statusOutputToggle[aria-expanded=false][aria-controls=statusOutputMessages]') !== null",'messages control did not start collapsed');
  await waitUI("document.querySelector('#statusOutput > [role=status][aria-atomic=true]')?.getClientRects().length === 1",'messages live region is unavailable when history is collapsed');
  await send('Runtime.evaluate',{expression:"document.querySelector('#statusOutputToggle').focus()"});
  for (const expanded of ['true','false']) {
    await send('Input.dispatchKeyEvent',{type:'keyDown',key:'Enter',code:'Enter',text:'\r',unmodifiedText:'\r',windowsVirtualKeyCode:13});
    await send('Input.dispatchKeyEvent',{type:'keyUp',key:'Enter',code:'Enter',windowsVirtualKeyCode:13});
    await waitUI(`document.activeElement.getAttribute('aria-expanded')==='${expanded}' && (getComputedStyle(document.querySelector('#statusOutputMessages')).display!=='none')===${expanded}`,'messages keyboard disclosure state did not match visibility');
  }
  await waitUI("Array.from(document.querySelectorAll('#sideMenu a')).some(a=>a.textContent==='Trip List'&&getComputedStyle(a).display!=='none')",'standard member Trip List menu item is hidden');
  const menuClose=await send('Runtime.evaluate',{expression:"document.querySelector('#menuIcon').click();const close=document.querySelector('#sideMenuClose');const valid=close?.textContent===String.fromCharCode(0x00d7)&&close.getAttribute('aria-label')==='Close menu'&&document.querySelector('#sideMenu').style.width==='250px';const route=location.hash;close.click();valid&&document.querySelector('#sideMenu').style.width==='0px'&&location.hash===route",returnByValue:true});
  assert.equal(menuClose.result?.value,true,'side menu close symbol or action is incorrect');
  const safeUserText=await send('Runtime.evaluate',{expression:"window.__tmXss!==true && !document.querySelector('#userDisplay img') && document.querySelector('#userDisplay').textContent.includes('<img')",returnByValue:true});
  assert.equal(safeUserText.result?.value,true,'stored profile markup executed or was interpreted as HTML');
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
  await send('Runtime.evaluate', {expression: "location.hash='/trips'"});
  let tripsRendered=false;
  for(let i=0;i<120;i++){
    const result=await send('Runtime.evaluate',{expression:"Array.from(document.querySelectorAll('[id=\"tripView.itemDiv\"]')).some(el=>el.getClientRects().length>0&&el.textContent.includes('Synthetic trip'))",returnByValue:true});
    if(result.result?.value===true){tripsRendered=true;break;}await delay(100);
  }
  assert(tripsRendered,'trip list did not render in the browser');
  const visibleName="Array.from(document.querySelectorAll('#itemName')).find(el=>el.getClientRects().length>0)";
  await send('Runtime.evaluate',{expression:"Array.from(document.querySelectorAll('[id=\"tripView.itemDiv\"] button')).find(button=>button.textContent==='Edit'&&button.getClientRects().length>0).click()"});
  let formReady=false;
  for(let i=0;i<120;i++){
    const result=await send('Runtime.evaluate',{expression:`!!(${visibleName}) && Array.from(document.querySelectorAll('#itemTripCostGroupID')).some(el=>el.getClientRects().length>0&&el.options.length>0)`,returnByValue:true});
    if(result.result?.value===true){formReady=true;break;}await delay(100);
  }
  assert(formReady,'trip edit form did not load');
  await waitUI("document.querySelector('label[for=tripDescription]')?.textContent === 'Description' && !!document.querySelector('textarea#tripDescription')", 'trip editor has no labelled Description field');
  await send('Runtime.evaluate',{expression:`window.__tmWriteFetch=window.fetch;window.__tmWriteFailures=0;window.fetch=(url,options)=>{if(String(url).includes('/trips/1')&&options?.method==='PUT'){window.__tmWriteFailures++;window.__tmTripDescription=JSON.parse(options.body).description;return Promise.resolve(new Response('synthetic conflict',{status:409}));}return window.__tmWriteFetch(url,options);};(${visibleName}).value='Preserve unsaved trip';document.querySelector('#tripDescription').value='Keep trip description';(${visibleName}).closest('form').dispatchEvent(new Event('submit',{bubbles:true,cancelable:true}));`});
  let writeFailed=false;
  for(let i=0;i<120;i++){
    const result=await send('Runtime.evaluate',{expression:`window.__tmWriteFailures===1&&document.body.textContent.includes('Change failed; please retry.')&&(${visibleName})?.value==='Preserve unsaved trip'&&window.__tmTripDescription==='Keep trip description'&&document.querySelector('#tripDescription')?.value==='Keep trip description'`,returnByValue:true});
    if(result.result?.value===true){writeFailed=true;break;}await delay(100);
  }
  assert(writeFailed,'failed trip write lost the edit form or reported success');
  await send('Runtime.evaluate',{expression:`(${visibleName}).closest('form').dispatchEvent(new Event('submit',{bubbles:true,cancelable:true}));`});
  for(let i=0;i<60;i++){const result=await send('Runtime.evaluate',{expression:'window.__tmWriteFailures',returnByValue:true});if(result.result?.value===2)break;await delay(100);}
  const retries=await send('Runtime.evaluate',{expression:'window.__tmWriteFailures',returnByValue:true});assert.equal(retries.result.value,2,'failed edit was not retryable');
  await send('Runtime.evaluate',{expression:"window.fetch=window.__tmWriteFetch;delete window.__tmWriteFetch"});
  await send('Runtime.evaluate',{expression:"document.querySelector('#trip-questions-1').click()"});
  await waitUI("!!document.querySelector('#mainContent form [name=kind]')",'trip organizer could not open booking questions');
  await send('Runtime.evaluate',{expression:`(() => {
    const form = document.querySelector('#mainContent form [name=kind]').closest('form');
    Array.from(form.querySelectorAll('button')).find(b=>b.textContent==='Add question').click();
    const fields = form.querySelectorAll('fieldset');
    const question = fields[fields.length-1];
    question.querySelector('[name=key]').value = 'equipment';
    question.querySelector('[name=label]').value = 'What equipment do you need?';
    form.requestSubmit();
  })()`});
  await waitUI("!document.querySelector('#mainContent form [name=kind]')",'trip organizer could not publish booking questions');
  // A failed list request must expose a working Retry action.
  await send('Runtime.evaluate',{expression:`window.__tmFetch=window.fetch;let failReport=true;window.fetch=(url,options)=>{if(String(url).includes('/tripsReport')&&failReport){failReport=false;return Promise.resolve(new Response('synthetic unavailable',{status:503}));}return window.__tmFetch(url,options);};`});
  await send('Runtime.evaluate', {expression: "location.hash='/tripsreport'"});
  let retryShown=false;
  for(let i=0;i<120;i++){
    const result=await send('Runtime.evaluate',{expression:"!!document.querySelector('#mainContent .tm-load-error button')",returnByValue:true});
    if(result.result?.value===true){retryShown=true;break;}await delay(100);
  }
  assert(retryShown,'failed report did not offer Retry');
  await send('Runtime.evaluate',{expression:"document.querySelector('#mainContent .tm-load-error button').click()"});
  let reportRendered=false;
  for(let i=0;i<120;i++){
    const result=await send('Runtime.evaluate',{expression:"!!document.querySelector('#mainContent .tm-trip-row')",returnByValue:true});
    if(result.result?.value===true){reportRendered=true;break;}await delay(100);
  }
  assert(reportRendered,'Trip List participant report did not render in the browser');
  await send('Runtime.evaluate',{expression:"const scheduleFilters=document.querySelector('.tm-schedule-filters');scheduleFilters.querySelectorAll('select')[1].value='1';scheduleFilters.requestSubmit();"});
  await waitUI("new URLSearchParams(location.search).get('type')==='1' && !!document.querySelector('#trip-book-1:enabled')",'Trip List did not apply the type filter');
  assert(requests.some(url=>url.includes('view=schedule')&&url.includes('month=2026-01')&&url.includes('type=1')),'combined month and type filters were not sent to the API');
  const disclosureIcon=await send('Runtime.evaluate',{expression:"Array.from(document.querySelectorAll('#mainContent span')).some(span=>span.textContent===String.fromCharCode(0x25b6))",returnByValue:true});
  assert.equal(disclosureIcon.result?.value,true,'disclosure arrow rendered as corrupted text');
  assert(!requests.some(url=>url.includes('trip_id=')),'report eagerly fetched participant lists');
  await send('Runtime.evaluate',{expression:"Array.from(document.querySelectorAll('[aria-controls^=trip-details-]')).find(b=>b.getClientRects().length>0).focus()"});
  await send('Input.dispatchKeyEvent',{type:'keyDown',key:'Enter',code:'Enter',text:'\r',unmodifiedText:'\r',windowsVirtualKeyCode:13});
  await send('Input.dispatchKeyEvent',{type:'keyUp',key:'Enter',code:'Enter',windowsVirtualKeyCode:13});
  await waitUI("document.activeElement.getAttribute('aria-expanded')==='true'",'Trip List keyboard disclosure failed');
  for(let i=0;i<120&&!requests.some(url=>url.includes('trip_id='));i++)await delay(100);
  assert(requests.some(url=>url.includes('trip_id=')&&url.includes('limit=50')),'bounded lazy roster request missing');
  await waitUI("!document.querySelector('[id^=trip-details-] [aria-busy=true]')",'participant page remained busy');
  await send('Runtime.evaluate',{expression:"window.__cancelFetch=window.fetch;window.fetch=(url,options)=>{if(String(url).includes('trip_id=')){return new Promise((resolve,reject)=>{window.__finishRoster=()=>resolve(new Response('[]',{headers:{'Content-Type':'application/json'}}));window.__rosterSignal=options.signal;options.signal.addEventListener('abort',()=>{window.__rosterAborted=true;reject(new DOMException('Cancelled','AbortError'));},{once:true});});}return window.__cancelFetch(url,options);};const toggle=Array.from(document.querySelectorAll('[aria-controls^=trip-details-]')).find(b=>b.getClientRects().length>0);toggle.click();toggle.click();"});
  await waitUI("!!window.__rosterSignal",'delayed participant request did not start');
  await send('Runtime.evaluate',{expression:"location.hash='/mybookings'"});
  await waitUI("window.__rosterAborted===true",'leaving Trip List did not abort its participant request');
  await send('Runtime.evaluate',{expression:"window.__finishRoster();window.fetch=window.__cancelFetch;delete window.__cancelFetch"});
  await send('Runtime.evaluate',{expression:"window.fetch=window.__tmFetch;delete window.__tmFetch"});
  await send('Runtime.evaluate', {expression: "location.hash='/mybookings'"});
  await waitForPage('Items 1–100');
  await send('Runtime.evaluate', {expression:`Array.from((${pager}).querySelectorAll('button')).find(button=>button.textContent==='Next').click()`});
  await waitForPage('Items 101–106');
  await send('Runtime.evaluate', {expression:`Array.from((${pager}).querySelectorAll('button')).find(button=>button.textContent==='Previous').click()`});
  await waitForPage('Items 1–100');
  assert(requests.some(url => url.startsWith(`${origin}${prefix}/auth/menuUser`)), 'frontend did not use its current origin/prefix');
  assert(!requests.some(url => url.includes('localhost:8086')), 'frontend retained a compiled localhost URL');
  await send('Runtime.evaluate',{expression:"location.hash='/tripsreport'"});
  await waitUI("document.querySelector('#trip-book-1')?.disabled===false",'Trip List booking action did not appear');
  await send('Runtime.evaluate',{expression:"document.querySelector('#trip-book-1').click()"});
  await waitUI("!!document.querySelector('#trip-details-1 form #itemVoucherCode') && !document.querySelector('#trip-details-1 form #itemBookingPrice')",'Trip List booking form did not offer an optional voucher with a calculated price');
  await send('Runtime.evaluate',{expression:"location.hash='/mybookings'"});
  await waitForPage('Items 1–100');
  const memberBooking = "Array.from(document.querySelectorAll('#mainContent [id=\"myBookingsView.itemDiv\"]')).find(row=>row.textContent.includes('Notes:Member booking'))";
  const myBookingStep = "document.querySelector('#mainContent [id=\"myBookingsView.itemEditDiv\"]')";
  const createBooking = "Array.from(document.querySelectorAll('#mainContent button')).find(button=>button.textContent==='Create Booking')";
  await send('Runtime.evaluate',{expression:"window.__tripFetch=window.fetch;window.fetch=(url,options)=>{if(new URL(url,location.href).pathname.endsWith('/myBookings/trips')&&!window.__retryTrips){return new Promise(resolve=>{window.__failTrips=()=>{window.__retryTrips=true;resolve(new Response('Unavailable',{status:503}));};});}return window.__tripFetch(url,options);};"});
  await send('Runtime.evaluate',{expression:`(${createBooking}).click()`});
  await waitUI(`!!window.__failTrips && (${myBookingStep})?.textContent.includes('Loading trips') && !(${myBookingStep})?.querySelector('select')`,'booking form opened before trips loaded');
  await send('Runtime.evaluate',{expression:"window.__failTrips()"});
  await waitUI(`(${myBookingStep})?.querySelector('.tm-load-error button')?.textContent==='Retry'`,'trip load failure did not show a retry');
  await send('Runtime.evaluate',{expression:`(${myBookingStep}).querySelector('.tm-load-error button').click()`});
  await waitUI(`(${myBookingStep})?.querySelector('#itemTripID')?.options.length>0`,'retry did not populate the Trip dropdown');
  await send('Runtime.evaluate',{expression:`window.fetch=window.__tripFetch;Array.from((${myBookingStep}).querySelectorAll('button')).find(button=>button.textContent==='Cancel').click()`});
  await waitUI(`!!(${memberBooking})?.querySelector('button')`,'voucher booking did not render');
  await send('Runtime.evaluate',{expression:`Array.from((${memberBooking}).querySelectorAll('button')).find(button=>button.textContent==='Continue booking').click()`});
  await send('Runtime.evaluate',{expression:`Array.from((${myBookingStep}).querySelectorAll('button')).find(button=>button.textContent==='Continue to review').click()`});
  await waitUI(`(${myBookingStep})?.textContent.includes('SMOKE-FREE') && (${myBookingStep})?.textContent.includes('NZ$0.00')`,'review did not apply the saved voucher');
  await send('Runtime.evaluate',{expression:`Array.from((${myBookingStep}).querySelectorAll('button')).find(button=>button.textContent==='Confirm booking').click()`});
  await waitUI(`(${memberBooking})?.textContent.includes('Status:Confirmed')`,'zero-total booking was not confirmed');
  await waitUI(`Array.from((${memberBooking})?.querySelectorAll('button')||[]).map(button=>button.textContent).join(',')==='View People'`,'confirmed booking still offered draft actions');
  await waitUI("document.querySelector('#statusOutput > [role=status]')?.textContent==='Booking confirmed.'",'booking result was not announced');
  assert(requests.some(url=>url.endsWith('/bookings/checkout/confirm/1')), 'zero-total confirmation request missing');
  assert(!requests.some(url=>url.includes('/bookings/checkout/create/')), 'zero-total booking opened Stripe checkout');
  const tripBooking = "document.querySelector('#trip-details-1 [id=\"bookingView.itemEditDiv\"]')";
  const bookingButton = (label) => `Array.from((${tripBooking})?.querySelectorAll('button')||[]).find(button=>button.textContent===${JSON.stringify(label)})`;
  async function clickBooking(label) {
    const button = bookingButton(label);
    await waitUI(`!!(${button}) && !(${button}).disabled`,`booking action missing: ${label}`);
    await send('Runtime.evaluate',{expression:`(${button}).click()`});
  }
  await send('Runtime.evaluate',{expression:"location.hash='/tripsreport'"});
  await waitUI("document.querySelector('#pageTitle')?.textContent==='Trip List' && document.querySelector('#trip-book-1')?.getClientRects().length>0 && !document.querySelector('#trip-book-1').disabled",'Trip List booking action did not reappear');
  await send('Runtime.evaluate',{expression:"document.querySelector('#trip-book-1').click()"});
  await waitUI(`(${tripBooking})?.querySelector('#itemNotes')?.getClientRects().length>0`,'new booking details did not appear');
  await clickBooking('Save and continue');
  await waitUI(`(${tripBooking})?.textContent.includes('Step 2 of 4')`,'saving did not advance to people');
  await clickBooking('Add person');
  const personInput = `(${tripBooking}).querySelector('#itemPerson')`;
  await waitUI(`!!${personInput}`,'participant questionnaire did not load');
  await waitUI(`(${tripBooking}).textContent.includes('What equipment do you need?')`,'published question did not appear on participant form');
  const typePerson = value => send('Runtime.evaluate',{expression:`(() => { const input = ${personInput}; input.focus(); input.value = ${JSON.stringify(value)}; input.dispatchEvent(new Event('input', {bubbles:true})); })()`});
  const selectPerson = async () => {
    await waitUI(`(${tripBooking}).querySelector('#itemPerson-matches [role=option]')?.textContent.includes('Other Member')`,'typing did not find a participant');
    for (const key of ['ArrowDown', 'Enter']) {
      await send('Input.dispatchKeyEvent',{type:'keyDown',key});
      await send('Input.dispatchKeyEvent',{type:'keyUp',key});
    }
    await waitUI(`${personInput}.value.includes('Other Member') && ${personInput}.getAttribute('aria-expanded')==='false' && ${personInput}.validity.valid`,'keyboard did not select the participant');
  };
  await typePerson('Other Member');
  await selectPerson();
  await typePerson('Other Mem');
  await waitUI(`!${personInput}.validity.valid`,'editing the name left the previous selection valid');
  await selectPerson();
  await send('Runtime.evaluate',{expression:`(() => {
    const form = ${personInput}.closest('form');
    const values = ['Other Member', '0200000000', 'Synthetic address', 'Test Contact', '0200000001', 'None'];
    form.querySelectorAll('.tm-participant-field input').forEach((field, i) => field.value = values[i]);
    const [membership, needsTransport, offersTransport] = form.querySelectorAll('.tm-participant-field select');
    membership.value = 'member';
    needsTransport.value = 'no';
    offersTransport.value = 'yes';
  })()`});
  await clickBooking('Submit');
  await waitUI(`(${tripBooking})?.textContent.includes('Other Member') && !(${tripBooking}).querySelector('#itemPerson')`,'participant was not saved');
  await clickBooking('Back to booking details');
  await waitUI(`(${tripBooking}).querySelector('#itemNotes')?.value===''`,'Back did not restore the booking with blank notes');
  await clickBooking('Save and continue');
  await clickBooking('Continue to review');
  await waitUI(`(${tripBooking})?.textContent.includes('Step 3 of 4') && (${tripBooking})?.textContent.includes('Other Member')`,'review did not show the saved participant');
  await clickBooking('Back to people');
  await clickBooking('Continue to review');
  await clickBooking('Continue to payment');
  await waitUI(`(${tripBooking})?.textContent.includes('Step 4 of 4')`,'payment step did not appear');
  await clickBooking('Back to review');
  await clickBooking('Continue to payment');
  await clickBooking('Pay for booking');
  await waitUI(`!!(${bookingButton('Back to review')}) && !(${bookingButton('Back to review')}).disabled`,'checkout failure did not offer navigation back to review');
  assert(requests.some(url=>url.endsWith('/bookings/checkout/create/108')), 'payment was not requested for the saved booking');
  assert.equal(exceptions.length, 0, 'browser runtime exception');
  await send('Runtime.evaluate',{expression:"location.hash='/triptype'"});
await waitUI("Array.from(document.querySelectorAll('[id=\"tripTypeView.itemDiv\"]')).some(e=>e.getClientRects().length>0)",'read-only catalog did not render');
const readonly=await send('Runtime.evaluate',{expression:"!Array.from(document.querySelectorAll('[id=\"tripTypeView.itemListDiv\"] [data-tm-action]')).some(b=>b.getClientRects().length>0)",returnByValue:true});assert.equal(readonly.result.value,true,'read-only user sees write actions');
const legacyLogout=await request(`${prefix}/auth/logout/`,'GET',undefined,cookie);
  assert.equal(legacyLogout.status,405,'GET logout still mutates session state');
  assert.equal((await request(`${prefix}/bookings`,'GET',undefined,cookie)).status,200,'GET logout revoked the session');
  const logout=await request(`${prefix}/auth/logout/`,'POST',undefined,cookie);
  assert.equal(logout.status,202,'session-owned POST logout failed');
  assert(logout.headers['set-cookie']?.some(value=>value.startsWith('session=;')&&value.includes('Max-Age=0')),'logout did not expire browser cookie');
  assert.equal((await request(`${prefix}/bookings`,'GET',undefined,cookie)).status,401,'revoked session remained valid');
  const limitedLogin = await request(`${prefix}/auth/verify-otp`, 'POST', {token: 'synthetic-booking-otp'});
  assert.equal(limitedLogin.status, 200, 'booking member OTP login failed');
  const limitedCookie = limitedLogin.headers['set-cookie'].find(value=>value.startsWith('session=')).split(';')[0];
  assert.equal((await request(`${prefix}/trips`, 'GET', undefined, limitedCookie)).status, 403);
  assert.equal((await request(`${prefix}/myBookings/trips`)).status, 401);
  const bookingTrips = await request(`${prefix}/myBookings/trips?limit=1`, 'GET', undefined, limitedCookie);
  assert.equal(bookingTrips.status, 200, 'booking member could not load trip choices');
  assert.deepEqual(Object.keys(JSON.parse(bookingTrips.body)[0]).sort(), ['from_date','id','to_date','trip_name']);
  await send('Network.setCookie',{name:'session',value:limitedCookie.slice('session='.length),url:origin,secure:true,httpOnly:true,sameSite:'Lax'});
  await send('Page.navigate',{url:`${origin}/?smoke=booking-member`});
  await waitUI("document.querySelector('#userDisplay')?.textContent.includes('Other Member')",'booking member UI did not load');
  await send('Runtime.evaluate',{expression:"location.hash='/mybookings'"});
  await waitUI(`!!(${createBooking}) && !(${createBooking}).disabled`,'booking member could not start a booking');
  await send('Runtime.evaluate',{expression:`(${createBooking}).click()`});
  await waitUI(`(${myBookingStep})?.querySelector('#itemTripID')?.selectedOptions[0]?.textContent.includes('Synthetic trip')`,'booking member could not select a trip');
  await send('Runtime.evaluate',{expression:`(${myBookingStep}).querySelector('#submitEditBtn').click()`});
  await waitUI(`(${myBookingStep})?.textContent.includes('Step 2 of 4')`,'booking member could not save the booking');
  const adminLogin=await request(`${prefix}/auth/verify-otp`,'POST',{token:'synthetic-admin-otp'});
  assert.equal(adminLogin.status,200);
  const adminCookie=adminLogin.headers['set-cookie'].find(value=>value.startsWith('session=')).split(';')[0];
  const groupsPath = `${prefix}/securityGroup`;
  const createdGroup = await request(groupsPath, 'POST', {name: 'Smoke concurrency group'}, adminCookie);
  assert.equal(createdGroup.status, 201, 'security group creation failed');
  const groupPath = `${groupsPath}/${JSON.parse(createdGroup.body)}`;
  const originalGroup = JSON.parse((await request(groupPath, 'GET', undefined, adminCookie)).body)[0];
  assert(originalGroup.version > 0, 'security group response omitted its version');
  const updatedGroup = await request(groupPath, 'PUT', {name: 'Updated smoke group', version: originalGroup.version}, adminCookie);
  assert.equal(updatedGroup.status, 200, 'current security group edit failed');
  assert(JSON.parse(updatedGroup.body).version > originalGroup.version, 'security group version did not advance');
  for (const payload of [{name: 'Missing version'}, {name: 'Stale version', version: originalGroup.version}]) {
    const conflict = await request(groupPath, 'PUT', payload, adminCookie);
    assert.equal(conflict.status, 409, 'security group edit should conflict');
    assert.equal(JSON.parse(conflict.body).error.code, 'stale_record');
  }
  assert.equal((await request(groupPath, 'DELETE', undefined, adminCookie)).status, 200);
  await send('Network.setCookie',{name:'session',value:adminCookie.slice('session='.length),url:origin,secure:true,httpOnly:true,sameSite:'Lax'});
  await send('Page.navigate',{url:`${origin}/`});
  async function waitUI(expression,message){for(let i=0;i<120;i++){const value=await send('Runtime.evaluate',{expression,returnByValue:true});if(value.result?.value===true)return;await delay(100);}const diagnostic=await send('Runtime.evaluate',{expression:"JSON.stringify({active:document.activeElement.outerHTML,disclosures:Array.from(document.querySelectorAll('.tm-disclosure')).map(e=>({html:e.outerHTML,visible:!!e.getClientRects().length,disabled:e.disabled})),writes:window.__accountWrites,invalid:Array.from(document.querySelectorAll('input,select,textarea')).filter(e=>e.getClientRects().length&&!e.validity.valid).map(e=>({id:e.id,message:e.validationMessage})),feedback:Array.from(document.querySelectorAll('.tm-feedback')).map(e=>e.textContent)})",returnByValue:true});assert.fail(message+' '+diagnostic.result?.value+' exceptions='+exceptions.join(','));}
  await waitUI("document.querySelector('#userDisplay')?.textContent.includes('Smoke Administrator')",'admin UI did not load');
  await send('Runtime.evaluate',{expression:"location.hash='/users'"});
  const deleteButton="Array.from(document.querySelectorAll('#mainContent button')).find(b=>b.textContent==='Delete'&&b.getClientRects().length>0)";
  await waitUI(`!!(${deleteButton})`,'account actions did not render');
  await send('Runtime.evaluate',{expression:`(${deleteButton}).focus();(${deleteButton}).click()`});
  await waitUI("!!document.querySelector('dialog[open] textarea')",'accessible reason dialog did not open');
  const dialogFocus=await send('Runtime.evaluate',{expression:"document.activeElement.tagName==='TEXTAREA'",returnByValue:true});
  assert.equal(dialogFocus.result.value,true,'dialog did not move focus');
  await send('Input.dispatchKeyEvent',{type:'keyDown',key:'Escape',code:'Escape',windowsVirtualKeyCode:27});
  await waitUI("!document.querySelector('dialog[open]')",'Escape did not cancel the dialog');
  const restoredFocus=await send('Runtime.evaluate',{expression:"document.activeElement.textContent==='Delete'",returnByValue:true});
  assert.equal(restoredFocus.result.value,true,'dialog did not restore focus');
  await send('Runtime.evaluate',{expression:"Array.from(document.querySelectorAll('#mainContent button')).find(b=>b.textContent==='Edit'&&b.getClientRects().length>0).click()"});
  await waitUI("!!document.querySelector('[id^=inlineChangeReason-]')",'account reason is not an inline field');
  await send('Runtime.evaluate',{expression:"Array.from(document.querySelectorAll('#mainContent button')).filter(b=>b.textContent==='Edit'&&b.getClientRects().length>0)[1].click()"});
  await waitUI("document.querySelectorAll('form[id^=inlineEditForm_]').length===2",'second account form did not open');
  await send('Runtime.evaluate',{expression:`window.__accountFetch=window.fetch;window.__accountWrites=0;window.fetch=(url,options)=>{if(String(url).includes('/users/')&&options?.method==='PUT'){window.__accountWrites++;window.__accountURL=String(url);window.__accountBody=JSON.parse(options.body);return Promise.resolve(new Response(JSON.stringify({error:{code:'stale_record',message:'Record changed; reload before saving'}}),{status:409,headers:{'Content-Type':'application/json'}}));}return window.__accountFetch(url,options);};const reason=document.querySelector('[id^=inlineChangeReason-]');reason.value='Browser regression';const form=reason.closest('form');form.querySelector('[id^=itemName-]').value='Preserved account edit';form.requestSubmit();`});
  await waitUI("window.__accountWrites===1 && document.body.textContent.includes('Record changed; reload before saving')",'local conflict feedback missing');
  const preserved=await send('Runtime.evaluate',{expression:"document.querySelector('[id^=inlineChangeReason-]').closest('form').querySelector('[id^=itemName-]').value==='Preserved account edit'&&!document.querySelector('[id^=inlineChangeReason-]').disabled",returnByValue:true});
  assert.equal(preserved.result.value,true,'failed account edit was lost or remained disabled');
  const isolated=await send('Runtime.evaluate',{expression:"window.__accountURL.endsWith('/users/'+document.querySelector('form[id^=inlineEditForm_]').id.split('_').pop())&&window.__accountBody.name==='Preserved account edit'",returnByValue:true});
  assert.equal(isolated.result.value,true,'submitting the first account form targeted another account');
  await waitUI("!!document.querySelector('[data-tm-reload]')",'conflict did not offer explicit reload');
  await send('Runtime.evaluate',{expression:"document.querySelector('[data-tm-reload]').click()"});
  await waitUI("!!document.querySelector('dialog[open]')",'discard confirmation missing');
  await send('Input.dispatchKeyEvent',{type:'keyDown',key:'Escape',code:'Escape',windowsVirtualKeyCode:27});
  await waitUI("!document.querySelector('dialog[open]')",'discard cancellation failed');
  const conflictPreserved=await send('Runtime.evaluate',{expression:"document.querySelector('[id^=itemName-]').value==='Preserved account edit'",returnByValue:true});
  assert.equal(conflictPreserved.result.value,true,'cancelled reload discarded edits');
  await send('Runtime.evaluate',{expression:"document.querySelector('[data-tm-reload]').click()"});
  await waitUI("!!document.querySelector('dialog[open]')",'reload confirmation did not reopen');
  await send('Runtime.evaluate',{expression:"document.querySelector('dialog[open] button[type=submit]').click()"});
  await waitUI("document.querySelectorAll('form[id^=inlineEditForm_]').length===0",'confirmed reload did not discard stale forms');
  const users = "Array.from(document.querySelectorAll('#mainContent .tm-disclosure')).filter(b=>b.textContent.includes('Smoke Member')||b.textContent.includes('Other Member'))";
  await waitUI(`${users}.length===2`,'member disclosures did not load');
  await send('Runtime.evaluate',{expression:`${users}.forEach(button=>button.click())`});
  const membershipEdits = "Array.from(document.querySelectorAll('[id=\"userSecurityGroupView.itemListDiv\"] button')).filter(b=>b.textContent==='Edit'&&b.getClientRects().length>0)";
  await waitUI(`${membershipEdits}.length===2`,'membership edit actions did not load');
  const membershipSelects = "Array.from(document.querySelectorAll('select[id^=\"userSecurityGroupView.membership_\"]'))";
  await send('Runtime.evaluate',{expression:`${membershipEdits}.forEach(button=>button.click())`});
  await waitUI(`${membershipSelects}.length===2 && ${membershipSelects}[0].id!==${membershipSelects}[1].id && ${membershipSelects}[1]===document.activeElement`,'membership editors did not use unique controls or focus the last opened select');
  await send('Runtime.evaluate',{expression:`${membershipSelects}[1].form.querySelector('button[id$=_cancel]').click()`});
  await waitUI(`${membershipSelects}.length===1 && document.activeElement===${membershipEdits}[1]`,'cancelling a membership form did not preserve the other form and restore focus');
  await send('Runtime.evaluate',{expression:"window.fetch=window.__accountFetch;location.hash='/securitygroup'"});
  const disclosure="Array.from(document.querySelectorAll('[id=\"securityGroupView.groupCard\"] .tm-disclosure')).find(b=>b.getClientRects().length>0)";
  await waitUI(`!!(${disclosure})`,'semantic disclosure missing');
  await send('Runtime.evaluate',{expression:`(${disclosure}).focus()`});
  await send('Input.dispatchKeyEvent',{type:'keyDown',key:'Enter',code:'Enter',text:'\r',unmodifiedText:'\r',nativeVirtualKeyCode:13,windowsVirtualKeyCode:13});
  await send('Input.dispatchKeyEvent',{type:'keyUp',key:'Enter',code:'Enter',windowsVirtualKeyCode:13});
  await waitUI("document.activeElement.getAttribute('aria-expanded')==='true'",'keyboard disclosure failed');
  await send('Emulation.setDeviceMetricsOverride',{width:390,height:844,deviceScaleFactor:1,mobile:true});
  const mobile=await send('Runtime.evaluate',{expression:"window.innerWidth===390&&!!document.querySelector('meta[name=viewport]')&&document.documentElement.scrollWidth<=window.innerWidth+1",returnByValue:true});
  assert.equal(mobile.result.value,true,'mobile viewport is not configured');
  assert.equal(exceptions.length,0,`browser exceptions: ${exceptions.join(', ')}`);
  console.log('Real-browser WASM, account forms, modal focus, keyboard and mobile checks passed');
} finally {
  socket?.close();
  try {
    browser.kill('SIGTERM');
  } catch (error) {
    if (!(error instanceof TypeError)) throw error;
  }
  await browser.status;
}
