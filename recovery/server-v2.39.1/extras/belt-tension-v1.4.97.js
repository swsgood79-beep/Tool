
// ---- Standard installation tension table (Table 10) + Unit mass (Table 1/2) ----
const DATA = {
  timing: {
    label: "타이밍 벨트 (MXL~XXH)",
    mode: "single",
    belts: {
      "MXL": {M:1.2, row:{3.2:2.9,4.8:5.1,6.4:7.6,7.9:10.3,9.5:11.8,12.7:16.4}},
      "XL":  {M:2.1, row:{6.4:13.7,7.9:19.6,9.5:24.5,12.7:34.5,19.1:55.0,25.4:75.1}},
      "L":   {M:3.1, row:{12.7:52.0,19.1:87.3,25.4:123.0,38.1:198.8,50.8:271.6}},
      "H":   {M:3.8, row:{19.1:222,25.4:312,38.1:486,50.8:668,76.2:1050,101.6:1030}},
      "XH":  {M:11.1, row:{50.8:909,76.2:1430,101.6:2020}},
      "XXH": {M:14.8, row:{50.8:1120,76.2:1750,101.6:2480,127.0:3190}}
    }
  },
  gthtd: {
    label: "GT & HTD 벨트",
    mode: "single",
    belts: {
      "1.5GT": {M:0.9, row:{1.5:2.3,2.0:3.4,2.5:4.4,3.0:5.6,4.0:7.8,6.0:12.6,9.0:19.6}},
      "2GT":   {M:1.3, row:{2.0:4.3,2.5:5.2,3.0:6.4,4.0:9.4,6.0:15.8,9.0:25.2,12:34.6,15:44.1}},
      "3GT":   {M:2.5, row:{4.0:20,6.0:29,9.0:44,12:59,15:74,20:98}},
      "3M":    {M:2.4, row:{4.0:20,6.0:29,9.0:44,12:59,15:74,20:98}},
      "5GT":   {M:4.0, row:{9.0:55,12:75.9,15:96,20:137,25:178,30:220,40:302}},
      "5M":    {M:3.8, row:{9.0:55,12:75.9,15:96,20:137,25:178,30:220,40:302}},
      "EV5GT": {M:4.0, row:{9.0:71,12:98,15:125,20:178,25:232,30:286,40:393}},
      "8YU":   {M:5.2, row:{20:235,25:294,30:343,40:500,50:637,55:701,60:765,70:892,85:1100}},
      "8M":    {M:6.1, row:{20:235,25:294,30:343,40:500,50:637,55:701,60:765,70:892,85:1100}},
      "EV8YU": {M:5.1, row:{15:210,20:285,25:350,30:410,40:600,50:760,55:836.7,60:920,70:1100,85:1300}},
      "14M":   {M:10.0, row:{30:441,40:618,50:775,55:902,60:1000,70:1190,85:1470,100:1790,115:2090}},
      "EV14M": {M:8.7, row:{30:560,40:740,50:963.5,55:1100,60:1200,70:1400,85:1800,100:2100,115:2500,130:2900,150:3400,170:3800}}
    }
  },
  polychain: {
    label: "Polychain GT Carbon 벨트",
    mode: "minmax",
    belts: {
      "8MGT":  {M:4.7, row:{12:{min:190,max:260},15:{min:240,max:330},20:{min:320,max:430},21:{min:340,max:460},25:{min:400,max:540},30:{min:480,max:650},36:{min:580,max:780},40:{min:640,max:870},50:{min:800,max:1080},60:{min:970,max:1300},62:{min:1000,max:1340},70:{min:1130,max:1520},80:{min:1290,max:1730}}},
      "14MGT": {M:8.0, row:{20:{min:820,max:1100},30:{min:1230,max:1670},37:{min:1520,max:2050},40:{min:1640,max:2220},50:{min:2050,max:2780},60:{min:2460,max:3330},68:{min:2790,max:3770},70:{min:2870,max:3890},80:{min:3280,max:4440},90:{min:3690,max:5000},100:{min:4100,max:5550},115:{min:4720,max:6380},125:{min:5130,max:6940},130:{min:5330,max:7220}}}
    }
  },
  mectrol: {
    label: "Mectrol 벨트 (장척 우레탄)",
    mode: "single",
    belts: {
      "T5 (아라미드/Kevlar)":  {M:2.0, row:{10:20,15:29,20:39,25:49,30:59,50:98,100:196}},
      "T5 (스틸심선/Steel)":   {M:2.2, row:{10:20,15:29,20:39,25:49,30:59,50:98,100:196}},
      "T10 (아라미드/Kevlar)": {M:3.6, row:{15:129,20:173,25:216,30:259,50:431,100:863}},
      "T10 (스틸심선/Steel)":  {M:4.3, row:{15:129,20:173,25:216,30:259,50:431,100:863}},
      "AT5 (단위중량 미기재)":  {M:null, row:{10:60,15:99,20:137,25:178,30:220,50:376,100:773}},
      "AT10": {M:5.6, row:{15:183,20:235,25:294,30:343,50:637,100:1301}},
      "AT20": {M:9.9, row:{50:775,100:1792}},
      "5M":   {M:4.1, row:{10:60,20:137,30:220,50:376}},
      "8M":   {M:5.9, row:{20:235,30:343,50:637,100:1301}},
      "14M":  {M:10.7, row:{40:618,55:902,85:1470,100:1792}},
      "8YU":  {M:5.2, row:{20:235,25:294,30:343,50:637,60:637,80:1020,100:1275}},
      "XL (아라미드/Kevlar)": {M:1.9, row:{12.7:114,19.1:172,25.4:229,50.8:457,101.6:914}},
      "XL (스틸심선/Steel)":  {M:2.1, row:{12.7:114,19.1:172,25.4:229,50.8:457,101.6:914}},
      "L (아라미드/Kevlar)":  {M:3.0, row:{12.7:165,19.1:248,25.4:330,50.8:660,101.6:1321}},
      "L (스틸심선/Steel)":   {M:3.5, row:{12.7:165,19.1:248,25.4:330,50.8:660,101.6:1321}},
      "H (아라미드/Kevlar)":  {M:3.2, row:{19.1:296,25.4:394,50.8:787,101.6:1575}},
      "H (스틸심선/Steel)":   {M:3.9, row:{19.1:296,25.4:394,50.8:787,101.6:1575}}
    }
  }
};

const catSel = document.getElementById('category');
const typeSel = document.getElementById('beltType');
const widthSel = document.getElementById('width');
const massValue = document.getElementById('massValue');
const tensionValue = document.getElementById('tensionValue');
const dataHint = document.getElementById('dataHint');
const calcBtn = document.getElementById('calcBtn');

Object.keys(DATA).forEach(k=>{
  const o=document.createElement('option'); o.value=k; o.textContent=DATA[k].label; catSel.appendChild(o);
});

function fillTypes(){
  const cat = DATA[catSel.value];
  typeSel.innerHTML='';
  Object.keys(cat.belts).forEach(k=>{
    const o=document.createElement('option'); o.value=k; o.textContent=k; typeSel.appendChild(o);
  });
  fillWidths();
}

function fillWidths(){
  const cat = DATA[catSel.value];
  const belt = cat.belts[typeSel.value];
  widthSel.innerHTML='';
  Object.keys(belt.row).forEach(w=>{
    const o=document.createElement('option'); o.value=w; o.textContent=w+' mm'; widthSel.appendChild(o);
  });
  updatePreview();
}

function updatePreview(){
  const cat = DATA[catSel.value];
  const belt = cat.belts[typeSel.value];
  const w = widthSel.value;
  const entry = belt.row[w];

  massValue.textContent = (belt.M===null? '미기재' : belt.M + ' g/mm×m');

  if(cat.mode==='minmax'){
    tensionValue.innerHTML = `MIN <b>${entry.min}</b> N &nbsp;~&nbsp; MAX <b>${entry.max}</b> N`;
  } else {
    tensionValue.innerHTML = `<b>${entry}</b> N`;
  }

  if(belt.M===null){
    dataHint.innerHTML = '<span class="warn">이 벨트는 매뉴얼에 단위중량이 기재되어 있지 않아 Hz 계산이 불가합니다. 기준장력(N) 참고용으로만 표시됩니다.</span>';
    calcBtn.disabled = true;
  } else {
    dataHint.textContent = '표에 없는 폭은 지원되지 않습니다(매뉴얼 기재값만 선택 가능).';
    calcBtn.disabled = false;
  }
  document.getElementById('resultCard').style.display='none';
}

catSel.addEventListener('change', fillTypes);
typeSel.addEventListener('change', fillWidths);
widthSel.addEventListener('change', updatePreview);
fillTypes();

function calc(){
  const cat = DATA[catSel.value];
  const belt = cat.belts[typeSel.value];
  const w = widthSel.value;
  const W = parseFloat(w);
  const M = belt.M;
  const S = parseFloat(document.getElementById('span').value);

  if(M===null){ alert('이 벨트는 단위중량이 없어 계산할 수 없습니다.'); return; }
  if(isNaN(S)||S<=0){ alert('스팬 길이를 올바르게 입력해 주세요.'); return; }

  const denom = 4 * M * W * S*S * 1e-9;
  const entry = belt.row[w];

  function freqOf(To){ return Math.sqrt(To/denom); }
  function fmt(v){ return v>=100? v.toFixed(1) : v.toFixed(2); }

  const resultCard = document.getElementById('resultCard');
  const out = document.getElementById('tensionOut');
  const bd = document.getElementById('breakdown');

  if(cat.mode==='minmax'){
    const fMin = freqOf(entry.min), fMax = freqOf(entry.max);
    out.innerHTML = `${fmt(fMin)} ~ ${fmt(fMax)}`;
    bd.innerHTML =
      `M = <code>${M} g/mm×m</code> · W = <code>${W} mm</code> · S = <code>${S} mm</code><br>` +
      `To(MIN) = <code>${entry.min} N</code> → f(MIN) = <b>${fmt(fMin)} Hz</b><br>` +
      `To(MAX) = <code>${entry.max} N</code> → f(MAX) = <b>${fmt(fMax)} Hz</b>`;
  } else {
    const f = freqOf(entry);
    out.textContent = fmt(f);
    bd.innerHTML =
      `M = <code>${M} g/mm×m</code> · W = <code>${W} mm</code> · S = <code>${S} mm</code> · To = <code>${entry} N</code><br>` +
      `f = √( To ÷ (4×M×W×S²×10⁻⁹) ) = <b>${fmt(f)} Hz</b>`;
  }
  resultCard.style.display='block';
  resultCard.scrollIntoView({behavior:'smooth', block:'nearest'});
}

document.getElementById('calcBtn').addEventListener('click',calc);
try{const t=localStorage.getItem('alpha-color-theme-v1');if(t==='light'||t==='dark')document.documentElement.dataset.theme=t}catch(_){}
