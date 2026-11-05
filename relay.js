(() => {
  'use strict';
  const pointerURL = 'https://raw.githubusercontent.com/swsgood79-beep/Tool/main/qr-endpoint.json';
  const key = new URLSearchParams(location.hash.slice(1)).get('k') || '';
  const title = document.getElementById('title');
  const message = document.getElementById('message');
  const retry = document.getElementById('retry');
  let busy = false;
  function error(heading, text) {
    title.textContent = heading; message.textContent = text;
    retry.hidden = false;
  }
  function endpoint(value) {
    if (typeof value !== 'string') return null;
    try {
      const u = new URL(value);
      if (u.protocol !== 'https:' || u.username || u.password || u.port || u.search || u.hash || u.pathname !== '/') return null;
      if (!/^[a-z0-9]+(?:-[a-z0-9]+)*\.trycloudflare\.com$/.test(u.hostname)) return null;
      if (value !== u.origin) return null;
      return u.origin;
    } catch (_) { return null; }
  }
  async function fetchTimed(url, options, milliseconds) {
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), milliseconds);
    try { return await fetch(url, {...options, signal: controller.signal}); }
    finally { clearTimeout(timer); }
  }
  async function connect() {
    if (busy) return;
    if (!/^[A-Za-z0-9_-]{43}$/.test(key)) {
      error('작업장 QR을 다시 스캔해 주세요.', '이 페이지 주소만으로는 접수할 수 없습니다. 관리자가 전달한 QR코드로 열어 주세요.');
      retry.hidden = true; return;
    }
    busy = true; retry.disabled = true;
    title.textContent = '작업장에 연결하고 있어요.';
    message.textContent = '회사 PC의 최신 접속 주소를 확인하고 있습니다.';
    try {
      const response = await fetchTimed(`${pointerURL}?t=${Date.now().toString(36)}`, {cache: 'no-store', credentials: 'omit', referrerPolicy: 'no-referrer'}, 10000);
      if (!response.ok) throw new Error('pointer');
      const state = await response.json();
      if (state.application !== 'wanryotalk' || state.version !== 1 || state.online !== true) {
        error('지금은 접수 연결이 꺼져 있어요.', '관리자에게 회사 PC의 인터넷 연결을 시작해 달라고 요청하세요.'); return;
      }
      const origin = endpoint(state.endpoint);
      if (!origin) { error('연결 정보를 확인해 주세요.', '관리자가 PC 프로그램에서 인터넷 연결을 다시 시작해야 합니다.'); return; }
      const probe = await fetchTimed(`${origin}/ready/${encodeURIComponent(key)}`, {cache: 'no-store', credentials: 'omit', referrerPolicy: 'no-referrer'}, 7000);
      if (!probe.ok || (await probe.json()).ready !== true) throw new Error('offline');
      title.textContent = '작업장으로 이동합니다.';
      message.textContent = '출하검사·세팅검사 접수 화면을 열고 있습니다.';
      location.replace(`${origin}/work/${encodeURIComponent(key)}`);
    } catch (_) {
      error('회사 PC 연결을 확인해 주세요.', 'PC가 꺼져 있거나 주소 갱신 중일 수 있습니다. 잠시 후 다시 연결해 주세요.');
    } finally {
      busy = false; retry.disabled = false;
    }
  }
  retry.addEventListener('click', connect);
  connect();
})();
