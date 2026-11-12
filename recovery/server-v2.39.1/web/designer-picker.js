(()=>{
 'use strict';
 const picker=document.getElementById('designer-picker');if(!picker)return;
 const form=picker.form,name=form.querySelector('[name=designer_name]'),phone=form.querySelector('[name=designer_phone]'),status=document.getElementById('designer-picker-status'),reload=document.getElementById('reload-designers');
 function choose(){const selected=picker.selectedOptions[0];if(!selected||!selected.value){status.textContent='이름과 전화번호를 직접 입력할 수 있습니다.';return}name.value=selected.dataset.name;phone.value=selected.dataset.phone;status.textContent=`${selected.dataset.name} 담당자의 이름과 전화번호를 입력했습니다.`}
 picker.addEventListener('change',choose);
 for(const field of [name,phone])field.addEventListener('input',()=>{picker.value='';status.textContent='직접 입력한 담당자 정보로 저장합니다.'});
 reload.addEventListener('click',async()=>{
  reload.disabled=true;const selected=picker.value;
  try{
   const response=await fetch('/designers.json',{credentials:'same-origin',cache:'no-store'});if(!response.ok)throw new Error();const people=await response.json();if(!Array.isArray(people))throw new Error();
   const options=[new Option('직접 입력 / 기존 정보 유지','')];
   for(const person of people){if(typeof person.ID!=='string'||typeof person.Name!=='string'||typeof person.Phone!=='string')continue;const option=new Option(`${person.Name} · ${person.Phone}`,person.ID);option.dataset.name=person.Name;option.dataset.phone=person.Phone;options.push(option)}
   picker.replaceChildren(...options);picker.value=selected;
   if(picker.value)choose();else status.textContent='명단을 새로 불러왔습니다. 담당자를 선택해 주세요. 직접 입력한 내용은 유지했습니다.';
  }catch(_){status.textContent='명단을 불러오지 못했습니다. 잠시 후 다시 눌러 주세요.'}finally{reload.disabled=false}
 });
})();
(()=>{const button=document.getElementById('register-designer');if(!button)return;button.addEventListener('click',async()=>{const form=button.form,name=form.querySelector('[name=designer_name]').value.trim(),phone=form.querySelector('[name=designer_phone]').value,status=document.getElementById('designer-picker-status');if(!name||!phone){status.textContent='등록할 담당자 이름과 전화번호를 입력하세요.';return}button.disabled=true;try{const res=await fetch('/designers',{method:'POST',credentials:'same-origin',headers:{'Content-Type':'application/x-www-form-urlencoded'},body:new URLSearchParams({csrf:form.querySelector('[name=csrf]').value,action:'create',name,phone})});const response=await fetch('/designers.json',{credentials:'same-origin',cache:'no-store'});if(!response.ok)throw new Error();const people=await response.json(),person=people.find(p=>p.Name===name&&p.Phone===phone.replace(/[ ()-]/g,''));if(!person)throw new Error();const picker=document.getElementById('designer-picker');let option=[...picker.options].find(o=>o.value===person.ID);if(!option){option=new Option(`${person.Name} · ${person.Phone}`,person.ID);option.dataset.name=person.Name;option.dataset.phone=person.Phone;picker.add(option)}picker.value=person.ID;picker.dispatchEvent(new Event('change'));status.textContent='명단에 등록했습니다. 수주 등록을 누르면 이 담당자로 저장합니다.'}catch(_){status.textContent='명단에 등록하지 못했습니다. 이름·전화번호와 중복 등록 여부를 확인하세요.'}finally{button.disabled=false}})})();
