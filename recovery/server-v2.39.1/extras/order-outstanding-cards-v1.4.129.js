(() => {
  'use strict';
  const config = document.getElementById('order-outstanding-config');
  if (!config) return;
  const endpoint = config.dataset.endpoint;
  const latest = new Map();
  const selector = '.work-item[data-project],.admin-project[data-status="active"][data-project]';
  const node = (tag, text) => {
    const element = document.createElement(tag);
    if (text !== undefined) element.textContent = text;
    return element;
  };
  const cards = () => [...document.querySelectorAll(selector)];
  const visible = card => !card.hidden && !card.closest('[hidden]') && card.getClientRects().length > 0;
  let refreshTimer;
  let refreshing = false;
  let frameQueued = false;

  function paint(card, value) {
    const button = card.querySelector('.order-outstanding-button');
    if (!button) return;
    const state = value?.state || 'pending';
    const hidden = state === 'ready' && value.count === 0;
    button.hidden = hidden;
    button.disabled = state === 'pending';
    button.dataset.state = state;
    const label = state === 'pending' ? '미입고 확인 중…' : state === 'error' ? '미입고 조회 재시도' : '미입고 현황 보기';
    if (button.textContent !== label) button.textContent = label;
    button.title = value?.error || '';
    button.setAttribute('aria-label', `${card.dataset.order} ${label}`);
    card.classList.toggle('has-order-outstanding', !hidden);
  }
  function install() {
    let added = false;
    const modern = document.documentElement.dataset.alphaDesign === 'modern';
    for (const card of cards()) {
      const admin = card.classList.contains('admin-project');
      const target = modern
        ? card.querySelector(admin ? '.e-admin-footer' : '.design-order-footer')
        : card.querySelector(admin ? ':scope > summary > div:first-child' : '.summary-status');
      if (!target) continue;
      let button = card.querySelector('.order-outstanding-button');
      if (!button) {
        button = node('button', '미입고 확인 중…');
        button.type = 'button';
        button.className = 'order-outstanding-button';
        button.setAttribute('aria-haspopup', 'dialog');
        button.addEventListener('click', event => {
          event.preventDefault();
          event.stopPropagation();
          open(card, button);
        });
        added = true;
      }
      const inspection = target.querySelector('.e-action,.expand-hint');
      if (inspection && (button.parentElement !== target || button.nextElementSibling !== inspection)) inspection.before(button);
      else if (!inspection && button.parentElement !== target) target.append(button);
      const value = latest.get(card.dataset.project);
      paint(card, value?.number === card.dataset.order ? value : null);
    }
    return added;
  }
  function schedule(delay = 0) {
    clearTimeout(refreshTimer);
    refreshTimer = setTimeout(refreshCards, delay);
  }
  async function refreshCards() {
    if (refreshing || document.hidden) { schedule(5000); return; }
    refreshing = true;
    let pending = false;
    try {
      const selected = cards().filter(visible);
      const unique = [...new Map(selected.map(card => [card.dataset.project, card])).values()];
      for (let offset = 0; offset < unique.length; offset += 10) {
        const group = unique.slice(offset, offset + 10);
        const ids = group.map(card => card.dataset.project).join(',');
        let values;
        try {
          const response = await fetch(`${endpoint}?projects=${encodeURIComponent(ids)}`, {cache: 'no-store'});
          if (!response.ok) throw Error('미입고 현황을 조회하지 못했습니다');
          const result = await response.json();
          if (!result.ok || !result.projects) throw Error('조회 결과를 확인할 수 없습니다');
          values = result.projects;
        } catch (error) {
          values = Object.fromEntries(group.map(card => [card.dataset.project, {state: 'error', error: error.message}]));
        }
        for (const card of group) {
          const id = card.dataset.project;
          let value = values[id] || {state: 'error', error: '수주 조회 결과를 확인할 수 없습니다'};
          if (value.number && value.number !== card.dataset.order) value = {state: 'error', error: '수주 정보가 변경됐습니다. 목록을 새로고침해 주세요'};
          latest.set(id, {...value, number: card.dataset.order});
          if (value.state === 'pending') pending = true;
        }
      }
      for (const card of cards()) {
        const value = latest.get(card.dataset.project);
        if (value?.number === card.dataset.order) paint(card, value);
      }
    } finally {
      refreshing = false;
      schedule(pending ? 1500 : 15000);
    }
  }

  const dialog = node('dialog');
  dialog.className = 'outstanding-dialog';
  dialog.setAttribute('aria-label', '수주 미입고 현황');
  const title = node('h2');
  const toolbar = node('div');
  toolbar.className = 'outstanding-toolbar';
  const refresh = node('button', '새로고침');
  const close = node('button', '닫기');
  refresh.type = close.type = 'button';
  toolbar.append(refresh, close);
  const stamp = node('p');
  const status = node('p');
  status.setAttribute('role', 'status');
  const content = node('div');
  dialog.append(title, toolbar, stamp, status, content);
  document.body.append(dialog);
  let selected;
  let opener;
  let generation = 0;
  let controller;
  let detailTimer;
  const date = value => /^\d{8}$/.test(value) ? `${value.slice(0,4)}-${value.slice(4,6)}-${value.slice(6)}` : value || '확인 불가';
  function render(rows, snapshot) {
    const groups = new Map();
    for (const row of rows) {
      const robot = row.robot || '로봇품번 확인 불가';
      if (!groups.has(robot)) groups.set(robot, []);
      groups.get(robot).push(row);
    }
    content.replaceChildren();
    const choices = new Map();
    let preview = null;
    const helper = node('section');
    helper.className = 'receipt-helper';
    const prepare = node('button', '선택 품목 ERP 입력 준비');
    const launch = node('button', '회사 PC 도우미 열기');
    const helperStatus = node('p', '입고된 품목과 수량을 선택해 주세요. ERP 저장은 직접 누릅니다.');
    const previewArea = node('div');
    prepare.type = launch.type = 'button';
    prepare.disabled = true; launch.hidden = true;
    const changed = () => {preview = null; launch.hidden = true; previewArea.replaceChildren(); prepare.disabled = choices.size === 0;};
    if (config.dataset.helperCsrf) {
      helper.append(node('h3', 'ERP 입고 입력 도우미'), helperStatus, prepare, launch, previewArea);
      content.append(helper);
    }
    async function requestHelper(start) {
      const values = [...choices].map(([key, quantity]) => ({key, quantity}));
      if (start && preview !== JSON.stringify(values)) {changed(); return;}
      prepare.disabled = launch.disabled = true;
      try {
        const response = await fetch('/erp-receipt-helper/prepare', {method: 'POST', body: new URLSearchParams({csrf:config.dataset.helperCsrf, project:selected.id, snapshot, selections:JSON.stringify(values), launch:start?'1':'0'})});
        if (!response.ok) throw Error(await response.text());
        const result = await response.json();
        if (!result.ok) throw Error('입력 준비를 완료하지 못했습니다');
        previewArea.replaceChildren();
        for (const group of result.task.groups) {
          const section = node('section');
          section.append(node('h4', `${group.customer} · 발주 ${group.purchase}`));
          for (const line of group.lines) {
            section.append(node('p', `${line.part || '품번 확인 불가'} /  ${line.item} · ${line.quantity}개 · 발주 행 ${line.sequence}`));
            if (!line.part && line.item_code) section.append(node('p',`ERP 품목 코드: ${line.item_code}`));
          }
          previewArea.append(section);
        }
        preview = JSON.stringify(values);
        launch.hidden = false;
        helperStatus.textContent = start ? '회사 PC 도우미가 열렸습니다. ERP에서 거래처를 확인한 뒤 발주 검색을 진행해 주세요. 체크·수량 입력·최종 저장은 직접 합니다.' : `${values.length}품목 준비 · 내용을 확인한 뒤 회사 PC 도우미를 열어 주세요.`;
      } catch(error) {helperStatus.textContent = error.message || '입력 준비에 실패했습니다';}
      finally {prepare.disabled = choices.size === 0; launch.disabled = false;}
    }
    prepare.addEventListener('click',()=>requestHelper(false));
    launch.addEventListener('click',()=>requestHelper(true));
    for (const [robot, items] of groups) {
      const section = node('section');
      const heading = node('h3', robot);
      const table = node('table');
      const head = node('thead');
      const labels = node('tr');
      for (const label of ['품명', '수량', '거래처', '입고예정일']) labels.append(node('th', label));
      head.append(labels);
      const body = node('tbody');
      for (const item of items) {
        const row = node('tr');
        let controlsRow;
        for (const value of [item.item || '확인 불가', item.quantity.toLocaleString('ko-KR'), item.customer || '확인 불가', date(item.due)]) row.append(node('td', value));
        if (config.dataset.helperCsrf) {
          const controls = node('div'); controls.className = 'receipt-choice';
          const label = node('label');
          const checkbox = node('input'); checkbox.type = 'checkbox'; checkbox.disabled = !item.key || !item.customer_code || !item.customer;
          checkbox.setAttribute('aria-label', `${item.part || item.item} 입고 선택`);
          label.append(checkbox, document.createTextNode('입고 선택'));
          const quantity = node('input'); quantity.type='number'; quantity.min='0.000001'; quantity.step='any'; quantity.max=String(item.quantity); quantity.value=String(item.quantity); quantity.disabled=true;
          quantity.setAttribute('aria-label', `${item.part || item.item} 입고 수량`);
          function syncChoice() {quantity.disabled=!checkbox.checked; if(checkbox.checked) choices.set(item.key,Number(quantity.value)); else choices.delete(item.key); changed();}
          checkbox.addEventListener('change',syncChoice); quantity.addEventListener('input',syncChoice);
          controls.append(label, quantity);
          if(item.part) {const partRow=node('tr');partRow.className='receipt-part';const cell=node('td',item.part);cell.colSpan=4;partRow.append(cell);body.append(partRow);}
          controlsRow=node('tr');const controlsCell=node('td');controlsCell.colSpan=4;controlsCell.append(controls);controlsRow.append(controlsCell);
          if (!item.key) controlsCell.append(node('small',item.prepare_block || '발주 행을 확인할 수 없어 선택할 수 없습니다'));
          else if (!item.customer_code || !item.customer) controlsCell.append(node('small','거래처 정보를 확인할 수 없어 선택할 수 없습니다'));
          if (!item.part && item.item_code) {const codeRow=node('tr');codeRow.className='receipt-part';const cell=node('td',`ERP 품목 코드: ${item.item_code} · 표시용 품번 미등록`);cell.colSpan=4;codeRow.append(cell);body.append(codeRow);}
        }
        body.append(row);
        if (controlsRow) body.append(controlsRow);
      }
      table.append(head, body);
      section.append(heading, table);
      content.append(section);
    }
  }
  function open(card, button) {
    selected = {id: card.dataset.project, number: card.dataset.order};
    opener = button;
    title.textContent = `수주 ${selected.number} · 미입고 현황`;
    if (!dialog.open) dialog.showModal();
    load(true);
  }
  function load(force) {
    ++generation;
    controller?.abort();
    clearTimeout(detailTimer);
    content.replaceChildren();
    stamp.textContent = '';
    status.textContent = '조회 중…';
    status.className = '';
    refresh.disabled = true;
    pollDetail(generation, force);
  }
  async function pollDetail(current, force) {
    controller = new AbortController();
    try {
      const response = await fetch(`${endpoint}?projects=${encodeURIComponent(selected.id)}&detail=1${force ? '&refresh=1' : ''}`, {cache: 'no-store', signal: controller.signal});
      if (!response.ok) throw Error('미입고 조회를 완료하지 못했습니다');
      const result = await response.json();
      if (current !== generation) return;
      const value = result.projects?.[selected.id];
      if (!result.ok || !value) throw Error('수주 조회 결과를 확인할 수 없습니다');
      if (value.number !== selected.number) throw Error('수주 정보가 변경됐습니다. 목록을 새로고침해 주세요');
      latest.set(selected.id, {...value, number: selected.number});
      for (const card of cards().filter(card => card.dataset.project === selected.id)) paint(card, value);
      if (value.state === 'pending') {
        detailTimer = setTimeout(() => pollDetail(current, false), 750);
        return;
      }
      if (value.state !== 'ready') throw Error(value.error || '미입고 조회에 실패했습니다');
      if (value.count > 0 && !Array.isArray(value.rows)) throw Error('미입고 품목을 확인할 수 없습니다');
      const rows = value.rows || [];
      status.textContent = rows.length ? `${rows.length}품목` : '미입고 없음';
      stamp.textContent = '조회 시각 ' + new Date(value.at).toLocaleString('ko-KR', {timeZone: 'Asia/Seoul'});
      render(rows, value.snapshot);
      refresh.disabled = false;
    } catch (error) {
      if (current !== generation || error.name === 'AbortError') return;
      status.className = 'outstanding-error';
      status.textContent = error.message || '조회에 실패했습니다. 새로고침해 주세요';
      refresh.disabled = false;
    }
  }
  refresh.addEventListener('click', () => load(true));
  close.addEventListener('click', () => dialog.close());
  dialog.addEventListener('close', () => {
    ++generation;
    controller?.abort();
    clearTimeout(detailTimer);
    if (opener?.isConnected && !opener.hidden) opener.focus({preventScroll: true});
  });
  new MutationObserver(() => {
    if (frameQueued) return;
    frameQueued = true;
    requestAnimationFrame(() => {
      frameQueued = false;
      if (install()) schedule(100);
    });
  }).observe(document.querySelector('main'), {childList: true, subtree: true});
  new MutationObserver(() => { install(); schedule(100); }).observe(document.documentElement, {attributes: true, attributeFilter: ['data-alpha-design']});
  new MutationObserver(() => schedule(100)).observe(document.body, {attributes: true, attributeFilter: ['data-design-tab']});
  document.addEventListener('input', event => {
    if (event.target.matches('#project-search,#admin-order-search')) schedule(300);
  });
  document.addEventListener('click', event => {
    if (event.target.closest('[data-filter],[data-order-filter],#order-prev,#order-next')) schedule(100);
  });
  document.addEventListener('visibilitychange', () => { if (!document.hidden) schedule(); });
  install();
  schedule();
})();
