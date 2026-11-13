(() => {
  const board = document.querySelector('main[data-board]')?.dataset.board;
  if (!board) return;
  const list = document.querySelector('.work-list');
  const search = document.getElementById('project-search');
  const clear = document.getElementById('clear-search');
  const filters = [...document.querySelectorAll('[data-filter]')];
  const emptySearch = document.getElementById('search-empty');
  const resultCount = document.getElementById('search-count');
  const todayPanel = document.getElementById('today-panel');
  const todayItems = document.getElementById('today-items');
  const todayEmpty = document.getElementById('today-empty');
  let selectedStage = 'all';
  let businessDate=document.getElementById('business-date').textContent;
  const dueUI=window.AlphaDue;
  const inStage=(card,stage)=>card.dataset.status!=='done'&&((stage==='all'?card.dataset.status!=='done':false)||(stage==='due_today'?dueUI.isDue(card.dataset.due,stage,businessDate):stage==='due_tomorrow'?dueUI.isDue(card.dataset.due,stage,businessDate):card.dataset.stage===stage));
  const normal = value => String(value).normalize('NFKC').toLocaleLowerCase('ko-KR').replace(/\s+/g, '');
  try {
    const saved = JSON.parse(sessionStorage.getItem('alpha-worker-view') || '{}');
    if (typeof saved.query === 'string') search.value = saved.query;
    if (['all','setting','shipping','today','due_today','due_tomorrow'].includes(saved.stage)) selectedStage = saved.stage;
  } catch (_) {}
  function filterCards() {
    const cards = sortCards();
    const query = normal(search.value);
    let shown = 0;
    for (const card of cards) {
      const matches = normal(card.dataset.order + ' ' + card.dataset.name).includes(query);
      const visible = matches && inStage(card,selectedStage) || !!card.querySelector('.worker-inspection-dialog[open]');
      card.hidden = !visible;
      if (visible) shown++;
    }
    const completed = [...todayItems.querySelectorAll('.today-item')];
    const viewingToday = selectedStage === 'today';
    list.hidden = viewingToday;todayPanel.hidden = !viewingToday;
    for (const card of completed) card.hidden = !normal(card.dataset.order + ' ' + card.dataset.name).includes(query);
    if (viewingToday) shown = completed.filter(card => !card.hidden).length;
    for (const button of filters) {
      button.setAttribute('aria-pressed', String(button.dataset.filter === selectedStage));
      button.querySelector('.filter-count').textContent = button.dataset.filter === 'today' ? completed.length : cards.filter(card => inStage(card,button.dataset.filter)).length;
    }
    dueUI.paint(document.querySelector('.delivery-panel'),cards,selectedStage,businessDate);
    dueUI.group(list,cards,selectedStage);
    clear.hidden = !search.value;
    const total = viewingToday ? completed.length : cards.filter(card=>inStage(card,selectedStage)).length;
    const heading = viewingToday ? '금일 검사 완료' : selectedStage==='due_today'?'오늘 납기':selectedStage==='due_tomorrow'?'다음 출고':'전체';
    resultCount.textContent = query || selectedStage !== 'all' ? `${heading} ${total}건 중 ${shown}건` : `전체 ${total}건`;
    document.getElementById('view-hint').textContent = viewingToday ? '오늘 검사를 모두 완료한 수주 목록입니다' : selectedStage==='due_today'?`${businessDate} 출고 예정 · 미완료 수주`:selectedStage==='due_tomorrow'?`${dueUI.label(dueUI.range(businessDate).end)}까지 출고 예정 · 미완료 수주`:'수주를 누르면 작업 화면이 열립니다';
    emptySearch.hidden = shown > 0 || (selectedStage==='all'&&total===0) || viewingToday;
    todayEmpty.hidden = completed.length > 0;
    try {sessionStorage.setItem('alpha-worker-view', JSON.stringify({query:search.value,stage:selectedStage}));} catch (_) {}
  }
  function sortCards() {return dueUI.order(list,[...list.querySelectorAll('.work-item,.delivery-completed')]);}
  search.addEventListener('input', filterCards);
  clear.addEventListener('click', () => {search.value=''; filterCards(); search.focus();});
  filters.forEach(button => button.addEventListener('click', () => {selectedStage=button.dataset.filter;filterCards();}));
  document.getElementById('reset-search').addEventListener('click', () => {search.value='';selectedStage='all';filterCards();search.focus();});
  filterCards();

  window.AlphaWorkerView = {refreshCard};
  async function refreshCard(project) {
    const current = list.querySelector(`[data-project="${project.id}"]`);
    if (current && current.dataset.revision === String(project.revision)) return;
    const response = await fetch(`/card/${encodeURIComponent(board)}/${project.id}`, {cache:'no-store'});
    if (!response.ok) return;
    const html = await response.text();
    const template = document.createElement('template'); template.innerHTML = html;
    const card = template.content.querySelector('.work-item,.delivery-completed');
    if (!card || card.dataset.project !== String(project.id)) return;
    if (current?.querySelector('details')?.open && card.querySelector('details')) card.querySelector('details').open = true;
    const wasOpen = !!current?.querySelector('.worker-inspection-dialog[open]');
    const keep = new Map();
    let focusKind = null, focusField = "worker", selection = null;
    if (current && current.dataset.generation === card.dataset.generation) {
      current.querySelectorAll('[data-inspection]').forEach(row => {
        const input = row.querySelector('input[name="worker"]');
        if (input) {
          keep.set(row.dataset.inspection, {worker:input.value, note:row.querySelector('textarea[name="note"]')?.value});
          const field = row.querySelector('textarea[name="note"]');
          if (document.activeElement === input || document.activeElement === field) {focusKind = row.dataset.inspection;focusField = document.activeElement === field ? "note" : "worker"; selection = document.activeElement.selectionStart;}
        }
      });
    }
    card.querySelectorAll('[data-inspection]').forEach(row => {
      const input = row.querySelector('input[name="worker"]');
      if (input && !input.matches(':disabled') && keep.has(row.dataset.inspection)) input.value = keep.get(row.dataset.inspection).worker;
      const note = row.querySelector('textarea[name="note"]');
      if (note && keep.has(row.dataset.inspection)) note.value = keep.get(row.dataset.inspection).note || "";
    });
    if (current) current.replaceWith(card); else list.appendChild(card);
    filterCards();
    if (wasOpen) window.AlphaWorkDialog?.open(card);
    if (focusKind) {
      const input = card.querySelector(`[data-inspection="${focusKind}"] [name="${focusField}"]`);
      if (input && !input.matches(':disabled')) {input.focus({preventScroll:true}); if (selection !== null) input.setSelectionRange(selection, selection);}
    }
  }
  function refreshShipments(shipments,date) {
    if (!Array.isArray(shipments) || typeof date !== 'string') return;
    document.getElementById('business-date').textContent = date;businessDate=date;
    const old = [...todayItems.querySelectorAll('.today-item')];
    if (old.length === shipments.length && old.every((card,i) => card.dataset.project === String(shipments[i].project_id) && card.dataset.order === shipments[i].number && card.dataset.name === shipments[i].name && card.dataset.info === (shipments[i].info_url || '') && card.dataset.photos === String(shipments[i].photo_count || 0) && card.dataset.settingWorker === (shipments[i].setting_worker || '') && card.dataset.shippingWorker === (shipments[i].shipping_worker || '') && card.dataset.suppliedMaterials === (shipments[i].supplied_materials||''))) return;
    const fragment = document.createDocumentFragment();
    for (const shipment of shipments) {
      if (typeof shipment.number !== 'string' || typeof shipment.name !== 'string') continue;
      const card = document.createElement('section');card.className = 'card today-item';
      card.dataset.project = String(shipment.project_id);card.dataset.order = shipment.number;card.dataset.name = shipment.name;card.dataset.info=shipment.info_url||'';card.dataset.photos=String(shipment.photo_count||0);
      card.dataset.settingWorker=shipment.setting_worker||'';card.dataset.shippingWorker=shipment.shipping_worker||'';card.dataset.suppliedMaterials=shipment.supplied_materials||'';
 const info = document.createElement('div');info.className = 'work-name';
      const number = document.createElement('span');number.className = 'pill order-number';number.textContent = `수주 ${shipment.number}`;
      const name = document.createElement('h2');name.textContent = shipment.name;
      const badge = document.createElement('span');badge.className = 'badge success';badge.textContent = '✓ 검사 완료';
      const top=document.createElement('div');top.className='project-topline';top.append(number);
      const infoPath=`/project-info/${encodeURIComponent(board)}/${shipment.project_id}`;
      if(shipment.supplied_materials){const b=document.createElement('button');b.type='button';b.className='supplied-materials';b.setAttribute('aria-haspopup','dialog');const label=document.createElement('b');label.textContent='사급품';const value=document.createElement('span');value.className='supplied-materials-text';value.textContent=shipment.supplied_materials;b.append(label,value);top.append(b)}
      if(shipment.info_url===infoPath){const link=document.createElement('a');link.className='special-info-link';link.href=infoPath;link.textContent='특이사항'+(shipment.photo_count>0?` ▧ ${shipment.photo_count}`:'');top.append(link);}
      info.append(top,name);const workers=document.createElement('div');workers.className='completed-workers';for(const [label,value] of [['세팅 작업자',shipment.setting_worker],['출하 작업자',shipment.shipping_worker]]){if(typeof value==='string'&&value){const line=document.createElement('p');line.textContent=label+' · '+value;workers.append(line)}}info.append(workers);card.append(info,badge);fragment.append(card);
    }
    todayItems.replaceChildren(fragment);
  }
  async function update() {
    try {
      const response = await fetch(`/status/${encodeURIComponent(board)}`, {cache:'no-store'});
      if (!response.ok) return;
      const state = await response.json();
      if (!Array.isArray(state.active) || !Array.isArray(state.stages)) return;
      refreshShipments(state.shipments,state.date);if(typeof state.schedule==='string')document.getElementById('schedule-hint').textContent=state.schedule;
      const completedDue=Array.isArray(state.completed_due)?state.completed_due:[];
 const active = new Set([...state.active.map(String),...completedDue.map(p=>String(p.id))]);
      list.querySelectorAll('.work-item[data-project],.delivery-completed[data-project]').forEach(card => {if (!active.has(card.dataset.project)) card.remove();});
      await Promise.all([...state.stages,...completedDue].map(refreshCard));
      filterCards();
      const empty = list.querySelector('.empty');
      if (empty) empty.hidden = state.active.length > 0 || !!list.querySelector('.delivery-completed:not([hidden])');
      if (!state.active.length && !empty && !list.querySelector('.delivery-completed:not([hidden])')) {
        const section = document.createElement('section'); section.className = 'card empty';
        const heading = document.createElement('h2'); heading.textContent = '현재 접수할 수주가 없습니다.';
        const note = document.createElement('p'); note.className = 'muted'; note.textContent = '모든 검사가 끝났거나 관리자가 목록을 정리했습니다.';
        section.append(heading, note); list.appendChild(section);
      }
    } catch (_) {
      // Temporary interruptions never erase input or completion records.
    } finally {setTimeout(update, 5000);}
  }
  setTimeout(update, 5000);
})();
