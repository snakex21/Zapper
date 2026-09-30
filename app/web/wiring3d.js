(() => {
  const canvas = document.getElementById('wiring3dCanvas');
  if (!canvas) return;
  const ctx = canvas.getContext('2d');
  if (!ctx) return;
  const scene = document.getElementById('wiring3dScene');
  let labelQueue = [];
  const font = 'Segoe UI, Arial, sans-serif';

  const state = {
    yaw: -0.34,
    pitch: 0.52,
    zoom: 0.82,
    panX: 0,
    panY: 0,
    dragging: false,
    panning: false,
    lastX: 0,
    lastY: 0,
    labels: false,
    autoFrame: true
  };

  const C = {
    bg: '#eef3f7', board: '#f8f7f1', boardEdge: '#b9c2c8', hole: '#566770',
    nano: '#176985', nanoEdge: '#084359', lcd: '#147250', lcdEdge: '#083d2a',
    screen: '#183f9a', ky: '#25383a', kyEdge: '#111e21', metal: '#b9c1c5',
    darkMetal: '#646d72', gold: '#d7ae3f', black: '#222', red: '#d32f2f',
    green: '#2e7d32', blue: '#1565c0', orange: '#f39c12', purple: '#8e24aa',
    cyan: '#00838f', out: '#ef6c00', text: '#17202a', white: '#fff'
  };

  const letters = ['A','B','C','D','E','F','G','H','I','J'];
  const leftPins = ['D13','3V3','REF','A0','A1','A2','A3','A4','A5','A6','A7','5V','RST','GND','VIN'];
  const rightPins = ['D12','D11','D10','D9','D8','D7','D6','D5','D4','D3','D2','GND','RST','RX0','TX1'];
  const pinRows = Object.fromEntries(leftPins.map((n,i)=>[n + '_L', i+2]).concat(rightPins.map((n,i)=>[n + '_R', i+2])));

  const B = { x:-170, y:-315, z:0, w:340, h:630 };
  const colX = { A:-140, B:-112, C:-84, D:-56, E:-28, F:28, G:56, H:84, I:112, J:140 };
  const rowY = r => B.y + 23 + (r-1) * 20;

  function v(x,y,z=0){ return {x,y,z}; }
  function add(a,b){ return v(a.x+b.x,a.y+b.y,a.z+b.z); }
  function rot(p){
    const cy=Math.cos(state.yaw), sy=Math.sin(state.yaw), cp=Math.cos(state.pitch), sp=Math.sin(state.pitch);
    let x=p.x*cy + p.z*sy;
    let z=-p.x*sy + p.z*cy;
    let y=p.y;
    const y2=y*cp - z*sp;
    const z2=y*sp + z*cp;
    return v(x,y2,z2);
  }
  function project(p){
    const r=rot(p);
    const f=900;
    const d=Math.max(260,1100 - r.z);
    const rect=canvas.getBoundingClientRect();
    // Automatyczne dopasowanie do mniejszych okien. Zoom użytkownika działa dalej ponad tym skalowaniem.
    const responsiveFit=Math.max(.1,Math.min(1,Math.min(rect.width/1180,rect.height/760)));
    const s=(f/d)*state.zoom*responsiveFit;
    return { x: rect.width/2 + state.panX + r.x*s, y: rect.height/2 + state.panY + r.y*s, z:r.z, s };
  }

  // Domyślny widok ma być faktycznie wyśrodkowany w całym polu, a nie tylko
  // względem samego breadboardu. LCD i KY-040 mocno poszerzają scenę, dlatego
  // liczymy ramkę całego montażu i dopiero z niej ustawiamy przesunięcie.
  function frameScene(){
    const rect=canvas.getBoundingClientRect();
    if(rect.width<2 || rect.height<2) return;

    state.panX=0;
    state.panY=0;

    const bounds={minX:-700,maxX:670,minY:-385,maxY:355,minZ:-80,maxZ:190};
    const points=[];
    for(const x of [bounds.minX,bounds.maxX]){
      for(const y of [bounds.minY,bounds.maxY]){
        for(const z of [bounds.minZ,bounds.maxZ]) points.push(project(v(x,y,z)));
      }
    }
    const minX=Math.min(...points.map(p=>p.x));
    const maxX=Math.max(...points.map(p=>p.x));
    const minY=Math.min(...points.map(p=>p.y));
    const maxY=Math.max(...points.map(p=>p.y));
    state.panX=(rect.width-(minX+maxX))/2;
    state.panY=(rect.height-(minY+maxY))/2;
  }

  function poly(points, fill, stroke, width=1){
    const q=points.map(project);
    ctx.beginPath(); ctx.moveTo(q[0].x,q[0].y); for(let i=1;i<q.length;i++) ctx.lineTo(q[i].x,q[i].y); ctx.closePath();
    if(fill){ctx.fillStyle=fill;ctx.fill();} if(stroke){ctx.strokeStyle=stroke;ctx.lineWidth=width;ctx.stroke();}
  }
  function line(points, color, width=3){
    const q=points.map(project); ctx.beginPath(); ctx.moveTo(q[0].x,q[0].y); for(let i=1;i<q.length;i++) ctx.lineTo(q[i].x,q[i].y);
    ctx.strokeStyle=color; ctx.lineWidth=width; ctx.lineCap='round'; ctx.lineJoin='round'; ctx.stroke();
  }
  function wire(points, color, width=4){
    const raised=points.map((p,i)=>i===0||i===points.length-1?p:v(p.x,p.y,p.z+Math.sin(i*Math.PI/(points.length-1))*34));
    const q=raised.map(project);
    // Rounded bends keep the exact terminals and lanes, while reading as insulated cable.
    const path=()=>{
      ctx.beginPath(); ctx.moveTo(q[0].x,q[0].y);
      for(let i=1;i<q.length-1;i++){
        const a=q[i-1],b=q[i],c=q[i+1];
        const r=Math.min(26,Math.hypot(b.x-a.x,b.y-a.y)/3,Math.hypot(c.x-b.x,c.y-b.y)/3);
        const ab=Math.hypot(b.x-a.x,b.y-a.y)||1,bc=Math.hypot(c.x-b.x,c.y-b.y)||1;
        ctx.lineTo(b.x+(a.x-b.x)*r/ab,b.y+(a.y-b.y)*r/ab);
        ctx.quadraticCurveTo(b.x,b.y,b.x+(c.x-b.x)*r/bc,b.y+(c.y-b.y)*r/bc);
      }
      ctx.lineTo(q[q.length-1].x,q[q.length-1].y);
    };
    ctx.save(); ctx.lineCap='round'; ctx.lineJoin='round';
    path();ctx.strokeStyle='rgba(36,53,63,.30)';ctx.lineWidth=width+2;ctx.shadowColor='rgba(28,47,63,.25)';ctx.shadowBlur=5;ctx.shadowOffsetY=5;ctx.stroke();
    ctx.shadowColor='transparent';path();ctx.strokeStyle=color;ctx.lineWidth=width+.6;ctx.stroke();
    ctx.translate(-.4,-.7);path();ctx.strokeStyle='rgba(255,255,255,.4)';ctx.lineWidth=1.2;ctx.stroke();ctx.restore();
  }
  function circle3(p, radius, fill, stroke, width=1){
    const q=project(p); const rr=Math.max(1,radius*q.s); ctx.beginPath(); ctx.arc(q.x,q.y,rr,0,Math.PI*2); if(fill){ctx.fillStyle=fill;ctx.fill();} if(stroke){ctx.strokeStyle=stroke;ctx.lineWidth=width;ctx.stroke();}
  }
  function label(p, text, opts={}){
    if(state.labels) labelQueue.push({p,text,opts});
  }
  function drawLabels(){
    const occupied=[],rect=canvas.getBoundingClientRect();
    // Remove labels from the second geometry pass, then place callouts only once.
    const seen=new Set();
    for(const {p,text,opts} of labelQueue){
      const key=text+JSON.stringify(p);if(seen.has(key))continue;seen.add(key);
      const q=project(p),size=Math.max(7,(opts.size||13)*Math.min(1,rect.width/950));
      if(q.x < -20 || q.x > rect.width+20 || q.y < 80 || q.y > rect.height+20)continue;
      ctx.font=`${opts.bold===false?'500':'600'} ${size}px ${font}`;
      const w=ctx.measureText(text).width+12,h=size+10;
      const desiredX=q.x+(opts.dx||0),desiredY=q.y+(opts.dy||0);
      let x=Math.max(w/2+8,Math.min(rect.width-w/2-8,desiredX)),y=desiredY;
      const collides=()=>occupied.some(r=>Math.abs(x-r.x)<(w+r.w)/2+3&&Math.abs(y-r.y)<(h+r.h)/2+3);
      // Compact row/column coordinates stay anchored; callouts can move with a leader.
      if(text.length>2){for(let i=0;i<20&&collides();i++)y=desiredY+(i%2?1:-1)*(Math.floor(i/2)+1)*(h+3);}
      y=Math.max(96+h/2,Math.min(rect.height-h/2-10,y));
      if(text.length>2) occupied.push({x,y,w,h});
      if(Math.abs(x-q.x)>22||Math.abs(y-q.y)>22){
        ctx.beginPath();ctx.moveTo(q.x,q.y);ctx.lineTo(x,y);ctx.strokeStyle=opts.stroke||'#9eafb9';ctx.lineWidth=.8;ctx.stroke();
        ctx.beginPath();ctx.arc(q.x,q.y,2,0,Math.PI*2);ctx.fillStyle=opts.stroke||'#8196a4';ctx.fill();
      }
      ctx.save();ctx.shadowColor='rgba(23,43,60,.09)';ctx.shadowBlur=5;ctx.shadowOffsetY=2;
      ctx.fillStyle=opts.bg||'rgba(255,255,255,.97)';ctx.strokeStyle=opts.stroke||'#d5dfe5';ctx.lineWidth=.8;
      roundRect(x-w/2,y-h/2,w,h,5);ctx.fill();ctx.shadowColor='transparent';ctx.stroke();ctx.restore();
      ctx.fillStyle=opts.color||C.text;ctx.textAlign='center';ctx.textBaseline='middle';ctx.fillText(text,x,y+.5);
    }
  }
  function silkText(p,text,opts={}){
    if(!state.labels){projectedText(p,text,(opts.size||8)/.65,opts.color||'#eef8fc',opts.align||'center');return;}
    const q=project(p); const size=(opts.size||8)*Math.min(1,Math.max(.55,q.s/.65));
    ctx.save();
    ctx.font=`800 ${size}px Consolas, monospace`;
    ctx.textAlign=opts.align||'center'; ctx.textBaseline='middle';
    ctx.lineWidth=1.4; ctx.strokeStyle='rgba(0,0,0,.22)'; ctx.strokeText(text,q.x+(opts.dx||0),q.y+(opts.dy||0));
    ctx.fillStyle=opts.color||'#eef8fc'; ctx.fillText(text,q.x+(opts.dx||0),q.y+(opts.dy||0));
    ctx.restore();
  }
  function roundRect(x,y,w,h,r){
    ctx.beginPath(); ctx.moveTo(x+r,y); ctx.arcTo(x+w,y,x+w,y+h,r); ctx.arcTo(x+w,y+h,x,y+h,r); ctx.arcTo(x,y+h,x,y,r); ctx.arcTo(x,y,x+w,y,r); ctx.closePath();
  }
  function box(cx,cy,cz,w,h,d,front,side){
    const bevel=Math.min(w,h)>80?6:Math.min(2,w*.12,h*.12);
    const x0=cx-w/2,x1=cx+w/2,y0=cy-h/2,y1=cy+h/2,z0=cz-d/2,z1=cz+d/2;
    const outline=[[x0+bevel,y0],[x1-bevel,y0],[x1,y0+bevel],[x1,y1-bevel],[x1-bevel,y1],[x0+bevel,y1],[x0,y1-bevel],[x0,y0+bevel]];
    const lower=outline.map(([x,y])=>v(x,y,z0)),upper=outline.map(([x,y])=>v(x,y,z1));
    const faces=[{p:lower,c:side||front,f:.73},{p:upper,c:front,f:1}];
    for(let i=0;i<8;i++)faces.push({p:[lower[i],lower[(i+1)%8],upper[(i+1)%8],upper[i]],c:side||front,f:.7+.22*(1+Math.cos(i*Math.PI/4))/2});
    faces.sort((a,b)=>a.p.reduce((s,p)=>s+rot(p).z,0)/a.p.length-b.p.reduce((s,p)=>s+rot(p).z,0)/b.p.length);
    for(const face of faces){
      const q=face.p.map(project),xs=q.map(p=>p.x),ys=q.map(p=>p.y);
      const grad=ctx.createLinearGradient(Math.min(...xs),Math.min(...ys),Math.max(...xs)+1,Math.max(...ys)+1);
      grad.addColorStop(0,shade(face.c,Math.min(1,face.f+.08)));grad.addColorStop(.45,shade(face.c,face.f));grad.addColorStop(1,shade(face.c,face.f*.78));
      poly(face.p,grad,'rgba(13,27,34,.3)',.65);
    }
    line([...upper,upper[0]],'rgba(255,255,255,.18)',.65);
  }

  function shade(hex,factor){
    if(!/^#[\da-f]{6}$/i.test(hex)) return hex;
    return '#'+hex.slice(1).match(/../g).map(c=>Math.round(parseInt(c,16)*factor).toString(16).padStart(2,'0')).join('');
  }
  function cylinder(cx,cy,z,r,depth,color){
    const ring=(height)=>Array.from({length:48},(_,i)=>v(cx+Math.cos(i*Math.PI/24)*r,cy+Math.sin(i*Math.PI/24)*r,height));
    const base=ring(z),top=ring(z+depth);
    for(let i=0;i<48;i++)poly([base[i],base[(i+1)%48],top[(i+1)%48],top[i]],shade(color,.62+.26*(1+Math.cos(i*Math.PI/24))/2),null);
    poly(top,color,'#657680',1);
    poly(ring(z+depth+.5).map(p=>v(cx+(p.x-cx)*.8,cy+(p.y-cy)*.8,p.z)),shade(color,.92),'#ffffff66',1);
  }
  function footprint(cx,cy,z,w,h){
    const corners=[v(cx-w/2,cy-h/2,z),v(cx+w/2,cy-h/2,z),v(cx+w/2,cy+h/2,z),v(cx-w/2,cy+h/2,z)];
    ctx.save();ctx.filter='blur(12px)';ctx.shadowColor='rgba(24,43,57,.20)';ctx.shadowBlur=24;ctx.shadowOffsetY=9;
    poly(corners,'rgba(30,49,62,.10)',null);ctx.restore();
  }
  function solder(p,r=6){
    circle3(p,r,'#b9c4c8','#64777e',.7);
    circle3(v(p.x-1,p.y-1,p.z+.1),r*.68,'#e3e9e8',null);
    circle3(v(p.x,p.y,p.z+.2),r*.36,'#596b72',null);
  }
  function smd(x,y,z,w=15,h=7){
    box(x,y,z,w+5,h,4,'#bac3bf','#64716e');
    box(x,y,z+1,w-3,h,5,'#c2aa79','#746543');
  }
  function circuitTrace(points,color='rgba(132,197,187,.19)'){
    line(points,color,1.1);
    for(const p of [points[0],points[points.length-1]])circle3(p,2.2,color,null);
  }
  function projectedText(p,text,size,color,align='center'){
    const q=project(p),x=project(v(p.x+1,p.y,p.z)),y=project(v(p.x,p.y+1,p.z));
    const facing=rot(v(0,0,1)).z>=0?1:-1;
    ctx.save();ctx.transform((x.x-q.x)*facing,(x.y-q.y)*facing,y.x-q.x,y.y-q.y,q.x,q.y);
    ctx.font=`600 ${size}px ${font}`;ctx.textAlign=align;ctx.textBaseline='middle';ctx.fillStyle=color;ctx.fillText(text,0,0);ctx.restore();
  }
  function drawBreadboard(){
    box(0,0,-14,B.w,B.h,28,C.board,C.boardEdge);
    // Moulded edge and individual recessed sockets, without changing the A–J grid.
    for(const x of [-164,164])line([v(x,-300,1),v(x,300,1)],'#ffffff',1.5);
    // central trench
    poly([v(-17,B.y+10,2),v(17,B.y+10,2),v(17,B.y+B.h-10,2),v(-17,B.y+B.h-10,2)], '#d9d6cb', '#bdb9ad');

    // Delikatnie zaznacz dwa wewnętrznie połączone rzędy F–J używane przez D9/rezystor.
    for(const r of [1,5]){
      poly([
        v(colX.F-12,rowY(r)-7,4),v(colX.J+12,rowY(r)-7,4),
        v(colX.J+12,rowY(r)+7,4),v(colX.F-12,rowY(r)+7,4)
      ], 'rgba(239,108,0,.12)', 'rgba(239,108,0,.38)');
    }
    // Czytelny ślad połączenia WEWNĄTRZ breadboardu: to nie jest dodatkowy kabel.
    line([v(colX.H,rowY(5),5),v(colX.I,rowY(5),5)],'rgba(239,108,0,.62)',5);
    line([v(colX.I,rowY(1),5),v(colX.J,rowY(1),5)],'rgba(239,108,0,.62)',5);

    // holes
    for(let r=1;r<=30;r++){
      for(const l of letters){
        const x=colX[l],y=rowY(r);
        poly([v(x-5,y-4,2),v(x+5,y-4,2),v(x+5,y+4,2),v(x-5,y+4,2)],'#bcc3c4','#fffdfa',.6);
        poly([v(x-2.7,y-2,3),v(x+2.7,y-2,3),v(x+2.7,y+2,3),v(x-2.7,y+2,3)],'#4d5b60',null);
      }
    }
    // labels
    for(const l of letters){
      projectedText(v(colX[l],B.y+8,5),l,11,'#7b8b90');
      label(v(colX[l],B.y-1,8),l,{size:11,bg:'rgba(255,255,255,.88)',dy:-7});
    }
    for(let r=1;r<=30;r+=1){
      if(r===1||r===2||r===5||r===9||r===10||r===11||r===12||r===13||r===15||r===16||r%5===0){
        projectedText(v(B.x+9,rowY(r),5),String(r),9,'#7b8b90');
        projectedText(v(B.x+B.w-9,rowY(r),5),String(r),9,'#7b8b90');
        label(v(B.x-12,rowY(r),8),String(r),{size:10,bg:'rgba(255,255,255,.84)',dx:-10});
        label(v(B.x+B.w+12,rowY(r),8),String(r),{size:10,bg:'rgba(255,255,255,.84)',dx:10});
      }
    }
    silkText(v(0,rowY(28),8),'A–J / 1–30',{size:9,color:'#657985'});

    // Otwory krytyczne dla rezystora/wyjścia. Etykiety rozstawione osobno, żeby się nie zlewały.
    circle3(v(colX.I,rowY(5),11),7,'#fff3e0',C.out,3);
    circle3(v(colX.I,rowY(1),11),7,'#fff3e0',C.out,3);
    circle3(v(colX.J,rowY(1),11),7,'#fff3e0',C.out,3);
    label(v(colX.I,rowY(5),18),'i5 · D9/H5',{size:9,bg:'#fff8e1',stroke:C.out,dx:78,dy:22});
    label(v(colX.I,rowY(1),18),'i1 · rezystor',{size:9,bg:'#fff8e1',stroke:C.out,dx:58,dy:-39});
    label(v(colX.J,rowY(1),18),'j1 · czerwony OUT',{size:9,bg:'#fff8e1',stroke:C.out,dx:126,dy:-10});
  }

  function drawBoardConnector(col,row,color,text,opts={}){
    const x=colX[col], y=rowY(row);
    // Goldpin wpięty w breadboard + kolorowa plastikowa tulejka jak na przewodzie Dupont.
    circle3(v(x,y,9),6.6,'#d9dde0','#667176',1.4);
    circle3(v(x,y,10),2.7,C.gold,'#80651e',1);
    box(x,y,23,6,6,27,C.gold,'#80651e');
    box(x,y,38,13,13,9,color,'#30363a');
    label(v(x,y,49),text,{size:8,bg:'#fff',stroke:color,dx:opts.dx||0,dy:opts.dy||0});
  }

  function drawBoardConnectors(){
    // Lewa połowa A–E: te otwory są połączone poziomo z pinami Nano w kolumnie D.
    drawBoardConnector('A',9,C.green,'a9 · SDA',{dx:-27});
    drawBoardConnector('A',10,C.blue,'a10 · SCL',{dx:-30});
    drawBoardConnector('A',13,C.red,'a13 · LCD VCC',{dx:-58,dy:-23});
    drawBoardConnector('B',13,C.red,'b13 · KY +',{dx:64,dy:32});
    drawBoardConnector('A',15,C.black,'a15 · LCD GND',{dx:-42});
    drawBoardConnector('B',15,C.black,'b15 · CZARNY',{dx:38,dy:14});

    // Prawa połowa F–J: te otwory są połączone poziomo z prawą listwą Nano w kolumnie H.
    drawBoardConnector('J',10,C.cyan,'j10 · SW',{dx:35});
    drawBoardConnector('J',11,C.purple,'j11 · DT',{dx:35});
    drawBoardConnector('J',12,C.orange,'j12 · CLK',{dx:38});
    drawBoardConnector('J',13,C.black,'j13 · KY GND',{dx:45});
  }

  function drawNano(){
    const cx=14, cy=rowY(9), z=25, w=148, h=315;
    box(cx,cy,z,w,h,14,C.nano,C.nanoEdge);
    for(const x of [colX.D,colX.H])box(x,cy,17,15,302,16,'#253239','#0b151c');
    for(let i=0;i<6;i++){
      const y=cy-88+i*25;
      circuitTrace([v(cx-46,y,z+8),v(cx-24,y,z+8),v(cx-10,y+14,z+8)],'rgba(77,175,186,.3)');
      circuitTrace([v(cx+48,y+10,z+8),v(cx+30,y+10,z+8),v(cx+17,y-3,z+8)],'rgba(77,175,186,.26)');
    }
    for(const x of [cx-50,cx+50])for(const y of [cy-140,cy+140])solder(v(x,y,z+9),5);
    for(let i=0;i<3;i++){smd(cx-12+i*17,cy-66,z+12,8,6);smd(cx+35,cy+84+i*16,z+12,10,6);}
    box(cx-23,cy+108,z+13,17,25,8,'#263238','#10191d');
    projectedText(v(cx,cy+148,z+10),'NANO',17,'#c9e1e4');
    // USB
    box(cx,rowY(1)+10,z+18,62,42,30,'#d7e0e4','#7b8e99');
    box(cx,rowY(1)-8,z+18,47,4,17,'#202e38','#101923');
    line([v(cx-22,rowY(1)-11,z+29),v(cx+22,rowY(1)-11,z+29)],'#f1f5f4',1.4);
    for(const dx of [-20,20])box(cx+dx,rowY(1)+17,z+34,7,12,1,'#899aa2','#71818b');
    label(v(cx,rowY(1)+10,z+36),'USB',{size:11,bg:'#dfe5e8'});
    // MCU and reset
    for(let i=0;i<8;i++)for(const dx of [-34,34])box(cx+dx,cy-11+i*9,z+15,13,4,4,'#c6d1d2','#60747c');
    for(let i=0;i<6;i++)for(const dy of [-24,64])box(cx-23+i*9,cy+dy,z+15,4,13,4,'#c6d1d2','#60747c');
    box(cx,cy+20,z+20,58,76,8,'#303a3c','#111b20');
    projectedText(v(cx,cy+15,z+25),'ATMEGA',8,'#93a2a3');
    projectedText(v(cx,cy+28,z+25),'328P',10,'#93a2a3');
    circle3(v(cx-20,cy-8,z+25),2,'#77888d',null);
    circle3(v(-28,cy+125,z+30),8,'#d7dde0','#657176',1);
    label(v(cx,cy+150,z+30),'ARDUINO NANO',{size:15,bg:'rgba(18,96,132,.9)',color:'#fff',stroke:'#0d587c'});
    // pins and pin names
    for(let i=0;i<15;i++){
      const r=i+2, y=rowY(r);
      // Nano ma rozstaw 0.6 cala; przy lewym rzędzie w D prawa listwa wypada w H.
      for(const x of [colX.D,colX.H]){solder(v(x,y,34),6);box(x,y,20,4,4,36,C.gold,'#897138');}
      silkText(v(colX.D+16,y,36),leftPins[i],{size:7,align:'left'});
      silkText(v(colX.H-16,y,36),rightPins[i],{size:7,align:'right'});
    }
  }

  function drawResistor(){
    const x=colX.I, y1=rowY(1), y5=rowY(5), z=28;

    // Nóżki naprawdę wchodzą w i5 oraz i1.
    line([v(x,y5,8),v(x,y5-18,z)],'#a8b9bf',3);
    line([v(x,y1+18,z),v(x,y1,8)],'#a8b9bf',3);
    box(x,(y1+y5)/2,z,22,50,20,'#d9bd8b','#6d4c41');
    for(const off of [-14,0,14]){
      line([v(x-11,(y1+y5)/2+off,z+11),v(x+11,(y1+y5)/2+off,z+11)],off===0?'#d32f2f':'#6d4c41',3);
    }
    label(v(x,(y1+y5)/2,z+34),'1 kΩ · i5 → i1',{size:10,bg:'#fff8e1',stroke:C.out,dx:96,dy:-28});

    // Czerwony krokodylek: wyjście z j1, wysoko nad KY-040.
    const redY=y1-48;
    wire([v(colX.J,y1,12),v(260,redY,58),v(420,redY,58)],C.out,5);
    label(v(310,redY,68),'j1 → czerwony krokodylek',{size:10,bg:'#fff3e0',stroke:C.out,dy:-16});
    drawClip(v(475,redY,58),C.red,'CZERWONY');

    // Czarny krokodylek: osobna dolna trasa, daleko pod KY-040.
    const blackY=rowY(28)+46;
    wire([v(colX.B,rowY(15),38),v(250,blackY,38),v(420,blackY,38)],C.black,5);
    label(v(305,blackY,48),'b15 → czarny krokodylek',{size:10,bg:'#fff',stroke:'#777',dy:-16});
    drawClip(v(475,blackY,38),C.black,'CZARNY');
  }

  function drawClip(p,color,text){
    // Sleeve begins at x-55, the unchanged wire terminal, with metal jaws beyond it.
    box(p.x-22,p.y,p.z,66,23,20,color,shade(color,.65));
    box(p.x+17,p.y,p.z,20,16,13,'#bcc8ce','#607782');
    poly([v(p.x+17,p.y-8,p.z+7),v(p.x+49,p.y-3,p.z+7),v(p.x+41,p.y+3,p.z+7),v(p.x+17,p.y+8,p.z+7)],'#d6dfe2','#607782',.8);
    poly([v(p.x+17,p.y-7,p.z-4),v(p.x+43,p.y-1,p.z-4),v(p.x+49,p.y+3,p.z-4),v(p.x+17,p.y+8,p.z-4)],'#8a9fa9','#4f6572',.8);
    line([v(p.x+22,p.y+2,p.z+1),v(p.x+44,p.y+1,p.z+1)],'#354b58',1);
    for(let i=0;i<4;i++)line([v(p.x-41+i*11,p.y-9,p.z+11),v(p.x-41+i*11,p.y+9,p.z+11)],'rgba(255,255,255,.15)',1);
    projectedText(v(p.x-23,p.y,p.z+11),text,8,'#fff');
  }

  function drawLCD(){
    const cx=-520, cy=-80, z=125, w=340, h=172;
    // Underside header leads remain connected to the same I2C terminals in front view.
    for(let i=0;i<4;i++){
      const px=cx+122-39+i*26,py=cy+6+56;
      box(px,py,z-35,5,5,58,C.gold,'#716034');
      box(px,py,z-59,13,15,14,'#2f3d42','#121f25');
    }
    box(cx,cy,z,w,h,12,C.lcd,C.lcdEdge);
    const frontVisible=rot(v(0,0,1)).z>=0;
    for(const dx of [-w/2+13,w/2-13]) for(const dy of [-h/2+13,h/2-13]) solder(v(cx+dx,cy+dy,z+8),6);
    for(let i=0;i<16;i++)solder(v(cx-110+i*14,cy-h/2+9,z+8),3.1);
    if(frontVisible){
      box(cx,cy-5,z+16,w-42,h-48,23,'#263746','#102331');
      poly([v(cx-w/2+28,cy-h/2+28,z+9),v(cx+w/2-28,cy-h/2+28,z+9),v(cx+w/2-28,cy+h/2-38,z+9),v(cx-w/2+28,cy+h/2-38,z+9)],'#1e4d95','#101f34',2);
      for(let row=0;row<2;row++)for(let col=0;col<16;col++){
        const x=cx-124+col*16,y=cy-36+row*31;
        poly([v(x,y,z+17),v(x+11,y,z+17),v(x+11,y+22,z+17),v(x,y+22,z+17)],'rgba(129,181,255,.12)',null);
      }
      projectedText(v(cx,cy+65,z+9),'LCD1602  ·  I2C',14,'#d0e7d7');
      label(v(cx,cy+46,z+14),'LCD1602 16×2',{size:13,bg:'rgba(23,134,77,.9)',color:'#fff',stroke:C.lcdEdge});
    }
    // back I2C adapter and header are visible when the model is turned around
    const ax=cx+122, ay=cy+6;
    if(!frontVisible){
      box(ax,ay,z-18,104,126,12,'#1c1f20','#050606');
      box(ax+24,ay-24,z-30,26,26,8,'#1565c0','#0d3c73');
      // drobne elementy i pola lutownicze backpacka — żeby tył nie był pustym prostokątem
      box(ax-20,ay-22,z-29,34,18,7,'#262b2e','#111');
      box(ax-28,ay+4,z-29,18,10,6,'#b7a46a','#6f623b');
      box(ax+2,ay+4,z-29,18,10,6,'#b7a46a','#6f623b');
      for(let s=0;s<8;s++){
        const sx=ax-42+s*12;
        circle3(v(sx,ay-50,z-25),3.7,'#cbd1d4','#6d7478',1.2);
        circle3(v(sx,ay-50,z-26),1.7,C.gold,'#80651e',.8);
      }
      silkText(v(ax,ay+28,z-32),'PCF8574 · I2C',{size:8,color:'#eef8fc'});
    }
    // Cztery prawdziwie wyglądające goldpiny: plastikowa listwa + lut + długi pin.
    const pinNames=['GND','VCC','SDA','SCL'];
    const pinColors=[C.black,C.red,C.green,C.blue];
    const pinPos=[];
    for(let i=0;i<4;i++){
      const py=ay+56, px=ax-39+i*26;
      if(!frontVisible){
        box(px,py,z-22,18,14,11,'#111415','#030404');
        circle3(v(px,py,z-14),7,'#cbd1d4','#687176',1.6);
        circle3(v(px,py,z-15),2.6,C.gold,'#80651e',1);
        box(px,py,z-40,6,6,50,C.gold,'#80651e');
        silkText(v(px,py-13,z-24),pinNames[i],{size:7,color:pinColors[i]});
      }
      pinPos.push(v(px,py,z-66));
      if(!frontVisible) label(v(px,py,z-69),pinNames[i],{size:9,bg:'#fff',stroke:pinColors[i],dy:20});
    }
    return { pinPos, pinNames };
  }

  function drawKY(){
    const cx=535, cy=-55, z=130, w=235, h=250;
    for(let i=0;i<5;i++){
      const px=cx-78+i*39,py=cy+104;
      box(px,py,z-37,5,5,60,C.gold,'#716034');
      box(px,py,z-61,14,16,14,'#2f3d42','#121f25');
    }
    box(cx,cy,z,w,h,14,C.ky,C.kyEdge);
    for(const dx of [-w/2+17,w/2-17])for(const dy of [-h/2+17,h/2-17])solder(v(cx+dx,cy+dy,z+8),7);
    const frontVisible=rot(v(0,0,1)).z>=0;
    if(frontVisible){
      box(cx,cy-32,z+25,135,125,35,'#b9c5c9','#617782');
      for(const dx of [-55,55])box(cx+dx,cy-32,z+45,9,81,5,'#8e9fa7','#516773');
      cylinder(cx,cy-32,z+43,46,12,'#c1cbd1');
      cylinder(cx,cy-32,z+55,25,48,'#c1cbd1');
      line([v(cx-16,cy-32,z+104),v(cx+16,cy-32,z+104)],'#6f838e',2);
      for(let i=0;i<3;i++)smd(cx-49+i*48,cy+57,z+13,20,10);
      projectedText(v(cx,cy+81,z+9),'KY-040',18,'#e0e8df');
      const faceNames=['GND','+','SW','DT','CLK'];
      for(let i=0;i<5;i++){
        const x=cx-78+i*39;solder(v(x,cy+104,z+9),5);
        projectedText(v(x,cy+91,z+10),faceNames[i],8,'#c4d8d1');
        circuitTrace([v(x,cy+102,z+8),v(x,cy+68,z+8),v(cx+(i-2)*19,cy+43,z+8)]);
      }
      label(v(cx,cy+65,z+18),'KY-040',{size:14,bg:'rgba(20,20,20,.9)',color:'#fff',stroke:'#444'});
    }
    // Tył KY-040: pola lutownicze, plastikowa listwa i wystające goldpiny.
    const names=['GND','+','SW','DT','CLK'];
    const pinColors=[C.black,C.red,C.cyan,C.purple,C.orange];
    const pins=[];
    if(!frontVisible){
      box(cx,cy+104,z-22,196,20,11,'#111415','#030404');
      // kilka małych elementów od strony lutów dla większego realizmu
      box(cx-52,cy+18,z-19,28,13,6,'#30363a','#111');
      box(cx+8,cy+18,z-19,22,12,6,'#b5a26d','#6b603f');
      box(cx+48,cy+18,z-19,22,12,6,'#b5a26d','#6b603f');
      silkText(v(cx,cy+55,z-24),'KY-040 · TYŁ PCB',{size:8,color:'#eef8fc'});
    }
    for(let i=0;i<5;i++){
      const px=cx-78+i*39, py=cy+104;
      if(!frontVisible){
        circle3(v(px,py,z-14),7.5,'#cbd1d4','#687176',1.6);
        circle3(v(px,py,z-15),2.8,C.gold,'#80651e',1);
        box(px,py,z-40,7,7,52,C.gold,'#80651e');
        silkText(v(px,py-15,z-24),names[i],{size:7,color:pinColors[i]});
      }
      pins.push(v(px,py,z-68));
      if(!frontVisible) label(v(px,py,z-71),names[i],{size:9,bg:'#fff',stroke:pinColors[i],dy:20});
    }
    if(!frontVisible) label(v(cx,cy-2,z-47),'PINY / LUTOWANIE OD TYŁU',{size:9,bg:'rgba(20,20,20,.88)',color:'#fff',stroke:'#555'});
    return { pins, names };
  }

  function drawWires(lcd,ky){
    // Przewody nie mają już etykiet na środku — pełna legenda jest pod modelem.
    // Dzięki temu napisy nie nachodzą na siebie przy obracaniu.
    const lcdTargets=[
      v(colX.A,rowY(15),38), // LCD GND -> a15
      v(colX.A,rowY(13),38), // LCD VCC -> a13
      v(colX.A,rowY(9),38),  // LCD SDA -> a9
      v(colX.A,rowY(10),38)  // LCD SCL -> a10
    ];
    const lcdColors=[C.black,C.red,C.green,C.blue];
    const lcdLaneX=[-310,-292,-274,-256];
    lcd.pinPos.forEach((p,i)=>{
      const stub=v(p.x,p.y+30+i*10,p.z);
      const lane=v(lcdLaneX[i],stub.y,50+i*4);
      const turn=v(lcdLaneX[i],lcdTargets[i].y,42+i*3);
      wire([p,stub,lane,turn,lcdTargets[i]],lcdColors[i],4);
    });

    const kyTargets=[
      v(colX.J,rowY(13),38), // KY GND -> j13
      v(colX.B,rowY(13),38), // KY + -> b13; przewód prowadzony przez okolice c13 i dalej w dół
      v(colX.J,rowY(10),38), // SW -> j10
      v(colX.J,rowY(11),38), // DT -> j11
      v(colX.J,rowY(12),38)  // CLK -> j12
    ];
    const kyColors=[C.black,C.red,C.cyan,C.purple,C.orange];
    const kyLaneX=[302,326,350,374,398];
    ky.pins.forEach((p,i)=>{
      const stub=v(p.x,p.y+34+i*12,p.z);
      const lane=v(kyLaneX[i],stub.y,52+i*4);
      if(i===1){
        // KY + jest WPIĘTY w b13. Sam kabel od b13 biegnie najpierw poziomo przez okolice c13,
        // tam skręca w dół, a dopiero później idzie do modułu KY-040.
        const lowY=rowY(24);
        wire([p,stub,lane,v(colX.C,lowY,48),v(colX.C,rowY(13),44),kyTargets[i]],kyColors[i],4);
      }else{
        const turn=v(kyLaneX[i],kyTargets[i].y,44+i*3);
        wire([p,stub,lane,turn,kyTargets[i]],kyColors[i],4);
      }
    });
  }

  function drawTitle(){
    const width=canvas.getBoundingClientRect().width;
    ctx.save();ctx.textAlign='left';ctx.textBaseline='top';
    ctx.fillStyle='#647c8c';ctx.font=`600 10px ${font}`;ctx.fillText('Z A P P E R  /  3 D',24,22);
    ctx.fillStyle='#203a4b';ctx.font=`600 ${width<500?16:20}px ${font}`;ctx.fillText('MONTAŻ · PŁYTKA + MODUŁY',24,42);
    ctx.fillStyle='#647c8c';ctx.font=`500 10px ${font}`;ctx.fillText('PINY LCD / KY-040: WIDOK OD TYŁU',24,68);
    ctx.strokeStyle='#cedce5';ctx.beginPath();ctx.moveTo(24,88);ctx.lineTo(width-24,88);ctx.stroke();ctx.restore();
  }

  function render(){
    const rect=canvas.getBoundingClientRect();
    if(rect.width<2 || rect.height<2) return;
    if(state.autoFrame) frameScene();
    const dpr=Math.min(window.devicePixelRatio||1,2);
    const cssW=rect.width, cssH=rect.height;
    const w=Math.max(1,Math.round(cssW*dpr)), h=Math.max(1,Math.round(cssH*dpr));
    if(canvas.width!==w||canvas.height!==h){ canvas.width=w; canvas.height=h; }
    ctx.setTransform(dpr,0,0,dpr,0,0);
    ctx.clearRect(0,0,cssW,cssH);
    const background=ctx.createRadialGradient(cssW*.45,cssH*.4,10,cssW*.5,cssH*.5,cssW*.75);
    background.addColorStop(0,'#fbfcfb');background.addColorStop(.65,'#eef2f3');background.addColorStop(1,'#d9e3e8');
    ctx.fillStyle=background;ctx.fillRect(0,0,cssW,cssH);
    labelQueue=[];
    footprint(12,16,-43,B.w+4,B.h+4);
    footprint(-498,-49,-40,340,172);
    footprint(557,-24,-40,235,250);
    drawBreadboard();
    drawNano();
    const lcd=drawLCD();
    const ky=drawKY();
    drawWires(lcd,ky);
    drawResistor();
    drawBoardConnectors();
    // Ponowne narysowanie modułów nad przewodami daje czytelne płytki i prawidłowe końcówki z tyłu.
    drawNano();
    drawLCD();
    drawKY();
    drawLabels();
    drawTitle();
  }

  function setView(name){
    state.autoFrame=false;
    for(const button of document.querySelectorAll('[data-wiring-view]')) button.setAttribute('aria-pressed',String(button.dataset.wiringView===name));
    if(name==='top'){ state.yaw=0; state.pitch=0; state.zoom=.82; state.panX=0; state.panY=10; }
    else if(name==='rear'){ state.yaw=Math.PI; state.pitch=0.34; state.zoom=.80; state.panX=0; state.panY=20; }
    else if(name==='lcdRear'){ state.yaw=2.55; state.pitch=.40; state.zoom=1.05; state.panX=0; state.panY=0;
      const target=project(v(-520,-80,125)),rect=canvas.getBoundingClientRect();
      state.panX=rect.width/2-target.x;state.panY=rect.height/2-target.y; }
    else if(name==='kyRear'){ state.yaw=-2.48; state.pitch=.40; state.zoom=1.05; state.panX=0; state.panY=0;
      const target=project(v(535,-55,130)),rect=canvas.getBoundingClientRect();
      state.panX=rect.width/2-target.x;state.panY=rect.height/2-target.y; }
    else {
      // Lekka perspektywa pokazuje grubość płytek; widok z góry pozostaje płaski.
      state.yaw=-.34; state.pitch=.52; state.zoom=.82; state.panX=0; state.panY=0; state.autoFrame=true;
    }
    render();
  }
  window.wiring3dView=setView;
  window.wiring3dReset=()=>setView('iso');
  window.wiring3dToggleLabels=()=>{state.labels=!state.labels;
    const button=document.getElementById('wiring3dLabels');
    if(button)button.setAttribute('aria-pressed',String(state.labels));
    render();};

  canvas.addEventListener('contextmenu',e=>e.preventDefault());
  canvas.addEventListener('pointerdown',e=>{
    state.autoFrame=false;
    state.dragging=true; state.panning=e.button===2||e.shiftKey; state.lastX=e.clientX; state.lastY=e.clientY; canvas.setPointerCapture(e.pointerId); canvas.classList.add('dragging'); e.preventDefault();
  });
  canvas.addEventListener('pointermove',e=>{
    if(!state.dragging)return; const dx=e.clientX-state.lastX,dy=e.clientY-state.lastY; state.lastX=e.clientX;state.lastY=e.clientY;
    if(state.panning||e.shiftKey){state.panX+=dx;state.panY+=dy;} else {state.yaw+=dx*.008;state.pitch=Math.max(-1.45,Math.min(1.45,state.pitch-dy*.008));}
    render(); e.preventDefault();
  });
  const end=e=>{state.dragging=false;state.panning=false;canvas.classList.remove('dragging');try{canvas.releasePointerCapture(e.pointerId);}catch(_){}};
  canvas.addEventListener('pointerup',end); canvas.addEventListener('pointercancel',end); canvas.addEventListener('lostpointercapture',end);
  canvas.addEventListener('wheel',e=>{if(!e.deltaY)return;state.autoFrame=false;state.zoom=Math.max(.55,Math.min(2.0,state.zoom*(e.deltaY<0?1.08:.92)));render();e.preventDefault();},{passive:false});
  canvas.addEventListener('dblclick',e=>{setView('iso');e.preventDefault();});
  canvas.addEventListener('keydown',e=>{
    if(!['ArrowLeft','ArrowRight','ArrowUp','ArrowDown','+','=','-','0'].includes(e.key))return;
    e.preventDefault();state.autoFrame=false;
    if(e.key==='0'){setView('iso');return;}
    if(e.key==='+'||e.key==='=')state.zoom=Math.min(2,state.zoom*1.08);
    else if(e.key==='-')state.zoom=Math.max(.55,state.zoom*.92);
    else if(e.shiftKey){state.panX+=e.key==='ArrowLeft'?-16:e.key==='ArrowRight'?16:0;state.panY+=e.key==='ArrowUp'?-16:e.key==='ArrowDown'?16:0;}
    else{state.yaw+=e.key==='ArrowLeft'?-.08:e.key==='ArrowRight'?.08:0;state.pitch=Math.max(-1.45,Math.min(1.45,state.pitch+(e.key==='ArrowUp'?.08:e.key==='ArrowDown'?-.08:0)));}
    render();
  });
  new ResizeObserver(render).observe(scene||canvas);
  render();
})();
