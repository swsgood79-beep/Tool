(() => {
  'use strict';
  const dialog = document.createElement('dialog');
  dialog.className = 'outstanding-dialog';
  dialog.setAttribute('aria-label', '수주 미입고 현황');
  const element = (tag, text) => {
    const node = document.createElement(tag);
    if (text !== undefined) node.textContent = text;
    return node;
  };
  const title = element('h2', '미입고 현황');
  const toolbar = element('div');
  toolbar.className = 'outstanding-toolbar';
  const refresh = element('button', '새로고침');
  const close = element('button', '닫기');
  refresh.type = close.type = 'button';
  const stamp = element('p');
  const status = element('p');
  status.setAttribute('role', 'status');
  const content = element('div');
  toolbar.append(refresh, close);
  dialog.append(title, toolbar, stamp, status, content);
  document.body.append(dialog);
  let order = '';
  let generation = 0;
  let controller;
  const date = value => /^\d{8}$/.test(value) ? `${value.slice(0,4)}-${value.slice(4,6)}-${value.slice(6)}` : value || '확인 불가';
  async function load() {
    const current = ++generation;
    controller?.abort();
    controller = new AbortController();
    title.textContent = `수주 ${order} · 미입고 현황`;
    status.textContent = '조회 중…';
    status.className = '';
    content.replaceChildren();
    stamp.textContent = '';
    refresh.disabled = true;
    try {
      const response = await fetch(`/machining/outstanding?order=${encodeURIComponent(order)}`, {cache: 'no-store', signal: controller.signal});
      const result = await response.json();
      if (current !== generation) return;
      if (!response.ok || !result.ok || !Array.isArray(result.rows)) throw Error(result.error || '미입고 조회를 완료하지 못했습니다');
      const groups = new Map();
      for (const row of result.rows) {
        const robot = row.robot || '로봇품번 확인 불가';
        if (!groups.has(robot)) groups.set(robot, []);
        groups.get(robot).push(row);
      }
      stamp.textContent = '조회 시각 ' + new Date(result.at).toLocaleString('ko-KR', {timeZone: 'Asia/Seoul'});
      status.textContent = result.rows.length ? `${result.rows.length}품목` : '미입고 없음';
      for (const [robot, rows] of groups) {
        const section = element('section');
        const heading = element('h3', robot);
        const table = element('table');
        const head = element('thead');
        const headings = element('tr');
        for (const label of ['품명', '수량', '거래처', '입고예정일']) headings.append(element('th', label));
        head.append(headings);
        const body = element('tbody');
        for (const row of rows) {
          const tr = element('tr');
          for (const value of [row.item || '확인 불가', row.quantity.toLocaleString('ko-KR'), row.customer || '확인 불가', date(row.due)]) tr.append(element('td', value));
          body.append(tr);
        }
        table.append(head, body);
        section.append(heading, table);
        content.append(section);
      }
    } catch (error) {
      if (current !== generation || error.name === 'AbortError') return;
      status.className = 'outstanding-error';
      status.textContent = error.message || '조회에 실패했습니다. 새로고침해 주세요';
    } finally {
      if (current === generation) refresh.disabled = false;
    }
  }
  document.addEventListener('click', event => {
    const button = event.target.closest('.outstanding-open');
    if (!button) return;
    order = button.dataset.order;
    dialog.showModal();
    load();
  });
  refresh.addEventListener('click', load);
  close.addEventListener('click', () => dialog.close());
  dialog.addEventListener('close', () => { ++generation; controller?.abort(); refresh.disabled = false; });
})();
