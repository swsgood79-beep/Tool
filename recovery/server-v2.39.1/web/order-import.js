(()=>{'use strict';if(document.querySelector('[data-import-running]')){setTimeout(()=>location.reload(),2000);return}
 const form=document.getElementById('import-confirm');if(!form)return;
 const rows=[...form.querySelectorAll('input[name=rows]:not(:disabled)')],all=document.getElementById('import-select-all'),count=document.getElementById('import-selected-count');
 function update(){const selected=rows.filter(row=>row.checked).length;count.textContent=`${selected}건 선택`;all.checked=selected>0&&selected===rows.length;all.indeterminate=selected>0&&selected<rows.length;form.querySelector('button[type=submit]').disabled=selected===0}
 all.addEventListener('change',()=>{rows.forEach(row=>row.checked=all.checked);update()});rows.forEach(row=>row.addEventListener('change',update));update();
 for(const kind of ['designer','client']){
  const dialog=document.getElementById(`import-${kind}-dialog`),picker=dialog.querySelector('select');let target=null,rowID=null;
  document.querySelectorAll(`[data-import-pick=${kind}]`).forEach(button=>button.addEventListener('click',()=>{rowID=button.dataset.row;target=form.elements.namedItem(`${kind}_${rowID}`);picker.value=target.value;if(picker.selectedIndex<0)picker.value='-pick';dialog.showModal()}));
  dialog.querySelector('form').addEventListener('submit',event=>{if(event.submitter?.value==='apply'&&target){target.value=picker.value;document.querySelector(`[data-${kind}-label="${rowID}"]`).textContent=picker.selectedOptions[0].textContent;const cell=target.closest('td');cell.querySelector('details')?.remove();if(picker.value!=='-pick')cell.querySelector('.import-choice-warning')?.remove();const tr=target.closest('tr');tr.lastElementChild.textContent=[...tr.querySelectorAll('input[type=hidden]')].some(n=>n.value==='-pick')?'연결 선택 필요':'등록 가능'}});
 }
})();
