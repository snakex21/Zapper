(() => {
  const canvas = document.getElementById('wiring3dCanvas');
  if (!canvas) return;
  const ctx = canvas.getContext('2d');
  const scene = document.getElementById('wiring3dScene');

  const state = {
    yaw: 0,
    pitch: 0,
    zoom: 0.82,
    panX: 0,
    panY: 0,
    dragging: false,
    panning: false,
    lastX: 0,
    lastY: 0,
    labels: true,
    autoFrame: true
  };

  const C = {
    bg: '#f5f8f9', board: '#f2f0e7', boardEdge: '#c9c6ba', hole: '#8f9aa0',
    nano: '#177cae', nanoEdge: '#0d587c', lcd: '#17864d', lcdEdge: '#0b5d35',
    screen: '#183f9a', ky: '#171a1b', kyEdge: '#050606', metal: '#b9c1c5',
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
    const responsiveFit=Math.max(.48,Math.min(1,Math.min(rect.width/1180,rect.height/760)));
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
    const q=points.map(project);
    ctx.beginPath(); ctx.moveTo(q[0].x,q[0].y); for(let i=1;i<q.length;i++) ctx.lineTo(q[i].x,q[i].y);
    ctx.strokeStyle='rgba(255,255,255,.88)'; ctx.lineWidth=width+4; ctx.lineCap='round'; ctx.lineJoin='round'; ctx.stroke();
    ctx.beginPath(); ctx.moveTo(q[0].x,q[0].y); for(let i=1;i<q.length;i++) ctx.lineTo(q[i].x,q[i].y);
    ctx.strokeStyle=color; ctx.lineWidth=width; ctx.lineCap='round'; ctx.lineJoin='round'; ctx.stroke();
  }
  function circle3(p, radius, fill, stroke, width=1){
    const q=project(p); const rr=Math.max(1,radius*q.s); ctx.beginPath(); ctx.arc(q.x,q.y,rr,0,Math.PI*2); if(fill){ctx.fillStyle=fill;ctx.fill();} if(stroke){ctx.strokeStyle=stroke;ctx.lineWidth=width;ctx.stroke();}
  }
  function label(p, text, opts={}){
    if(!state.labels) return;
    const q=project(p); const size=opts.size||13; ctx.font=`${opts.bold===false?'500':'700'} ${size}px Segoe UI, Arial`;
    const pad=5, w=ctx.measureText(text).width + pad*2, h=size+8;
    let x=q.x + (opts.dx||0), y=q.y + (opts.dy||0);
    ctx.fillStyle=opts.bg||'rgba(255,255,255,.94)'; ctx.strokeStyle=opts.stroke||'#cfd8dc'; ctx.lineWidth=1;
    roundRect(x-w/2,y-h/2,w,h,5); ctx.fill(); ctx.stroke();
    ctx.fillStyle=opts.color||C.text; ctx.textAlign='center'; ctx.textBaseline='middle'; ctx.fillText(text,x,y+0.5);
  }
  function silkText(p,text,opts={}){
    if(!state.labels) return;
    const q=project(p); const size=opts.size||8;
    ctx.save();
    ctx.font=`800 ${size}px Consolas, monospace`;
    ctx.textAlign=opts.align||'center'; ctx.textBaseline='middle';
    ctx.lineWidth=2.4; ctx.strokeStyle='rgba(0,0,0,.42)'; ctx.strokeText(text,q.x+(opts.dx||0),q.y+(opts.dy||0));
    ctx.fillStyle=opts.color||'#eef8fc'; ctx.fillText(text,q.x+(opts.dx||0),q.y+(opts.dy||0));
    ctx.restore();
  }
  function roundRect(x,y,w,h,r){
    ctx.beginPath(); ctx.moveTo(x+r,y); ctx.arcTo(x+w,y,x+w,y+h,r); ctx.arcTo(x+w,y+h,x,y+h,r); ctx.arcTo(x,y+h,x,y,r); ctx.arcTo(x,y,x+w,y,r); ctx.closePath();
  }
  function box(cx,cy,cz,w,h,d,front,side){
    const x0=cx-w/2,x1=cx+w/2,y0=cy-h/2,y1=cy+h/2,z0=cz-d/2,z1=cz+d/2;
    const faces=[
      {p:[v(x0,y0,z0),v(x1,y0,z0),v(x1,y1,z0),v(x0,y1,z0)],c:side||front},
      {p:[v(x0,y0,z1),v(x1,y0,z1),v(x1,y1,z1),v(x0,y1,z1)],c:front},
      {p:[v(x0,y0,z0),v(x0,y1,z0),v(x0,y1,z1),v(x0,y0,z1)],c:side||front},
      {p:[v(x1,y0,z0),v(x1,y1,z0),v(x1,y1,z1),v(x1,y0,z1)],c:side||front},
      {p:[v(x0,y0,z0),v(x1,y0,z0),v(x1,y0,z1),v(x0,y0,z1)],c:side||front},
      {p:[v(x0,y1,z0),v(x1,y1,z0),v(x1,y1,z1),v(x0,y1,z1)],c:side||front}
    ];
    faces.sort((a,b)=>a.p.reduce((s,p)=>s+rot(p).z,0)/a.p.length - b.p.reduce((s,p)=>s+rot(p).z,0)/b.p.length);
    for(const face of faces) poly(face.p,face.c,'#0000002b');
  }

  function drawBreadboard(){
    box(0,0,-5,B.w,B.h,10,C.board,C.boardEdge);
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
      for(const l of letters){ circle3(v(colX[l],rowY(r),6),3.2,C.hole,'#69757a',0.5); }
    }
    // labels
    for(const l of letters) label(v(colX[l],B.y-1,8),l,{size:11,bg:'rgba(255,255,255,.88)',dy:-7});
    for(let r=1;r<=30;r+=1){
      if(r===1||r===2||r===5||r===9||r===10||r===11||r===12||r===13||r===15||r===16||r%5===0){
        label(v(B.x-12,rowY(r),8),String(r),{size:10,bg:'rgba(255,255,255,.84)',dx:-10});
        label(v(B.x+B.w+12,rowY(r),8),String(r),{size:10,bg:'rgba(255,255,255,.84)',dx:10});
      }
    }
    silkText(v(0,rowY(19),8),'PŁYTKA STYKOWA 400 PÓL · A–J / 1–30',{size:10,color:'#5f676b'});

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
    box(cx,cy,z,w,h,18,C.nano,C.nanoEdge);
    // USB
    box(cx,rowY(1)+10,z+18,62,42,30,'#cfd8dc','#8c979c');
    label(v(cx,rowY(1)+10,z+36),'USB',{size:11,bg:'#dfe5e8'});
    // MCU and reset
    box(cx,cy+20,z+20,58,76,8,'#20272b','#111');
    circle3(v(-28,cy+125,z+30),8,'#d7dde0','#657176',1);
    label(v(cx,cy+150,z+30),'ARDUINO NANO',{size:15,bg:'rgba(18,96,132,.9)',color:'#fff',stroke:'#0d587c'});
    // pins and pin names
    for(let i=0;i<15;i++){
      const r=i+2, y=rowY(r);
      // Nano ma rozstaw 0.6 cala; przy lewym rzędzie w D prawa listwa wypada w H.
      box(colX.D,y,15,10,10,30,C.metal,C.darkMetal);
      box(colX.H,y,15,10,10,30,C.metal,C.darkMetal);
      silkText(v(colX.D+16,y,36),leftPins[i],{size:7,align:'left'});
      silkText(v(colX.H-16,y,36),rightPins[i],{size:7,align:'right'});
    }
  }

  function drawResistor(){
    const x=colX.I, y1=rowY(1), y5=rowY(5), z=28;

    // Nóżki naprawdę wchodzą w i5 oraz i1.
    line([v(x,y5,8),v(x,y5-18,z)],C.out,3);
    line([v(x,y1+18,z),v(x,y1,8)],C.out,3);
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
    const q=project(p); const s=Math.max(.55,q.s);
    ctx.save(); ctx.translate(q.x,q.y); ctx.scale(s,s);
    ctx.fillStyle=color; ctx.strokeStyle='#111'; ctx.lineWidth=2;
    ctx.beginPath(); ctx.moveTo(-34,-10); ctx.lineTo(25,-14); ctx.lineTo(48,0); ctx.lineTo(25,14); ctx.lineTo(-34,10); ctx.closePath(); ctx.fill(); ctx.stroke();
    ctx.fillStyle='#fff'; ctx.font='700 9px Segoe UI'; ctx.textAlign='center'; ctx.textBaseline='middle'; ctx.fillText(text,2,0); ctx.restore();
  }

  function drawLCD(){
    const cx=-520, cy=-80, z=125, w=340, h=172;
    box(cx,cy,z,w,h,12,C.lcd,C.lcdEdge);
    const frontVisible=rot(v(0,0,1)).z>=0;
    if(frontVisible){
      poly([v(cx-w/2+28,cy-h/2+28,z+9),v(cx+w/2-28,cy-h/2+28,z+9),v(cx+w/2-28,cy+h/2-38,z+9),v(cx-w/2+28,cy+h/2-38,z+9)],C.screen,'#101418',4);
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
    box(cx,cy,z,w,h,12,C.ky,C.kyEdge);
    const frontVisible=rot(v(0,0,1)).z>=0;
    if(frontVisible){
      circle3(v(cx,cy-32,z+34),52,'#8f989d','#50595e',5);
      circle3(v(cx,cy-32,z+48),27,'#b7bec2','#626a6e',3);
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
    ctx.save();
    ctx.fillStyle='rgba(255,255,255,.92)'; ctx.strokeStyle='#d5dee2'; ctx.lineWidth=1;
    roundRect(18,18,390,66,10); ctx.fill(); ctx.stroke();
    ctx.fillStyle=C.text; ctx.textAlign='left'; ctx.textBaseline='top'; ctx.font='800 19px Segoe UI, Arial'; ctx.fillText('MONTAŻ 3D — PŁYTKA + PINY OD TYŁU',34,31);
    ctx.font='600 12px Segoe UI, Arial'; ctx.fillStyle='#52666f'; ctx.fillText('Przeciągnij: obrót · rolka: zoom · prawy/Shift: przesunięcie',34,58);
    ctx.restore();
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
    ctx.fillStyle=C.bg;
    ctx.fillRect(0,0,cssW,cssH);
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
    drawTitle();
  }

  function setView(name){
    state.autoFrame=false;
    if(name==='top'){ state.yaw=0; state.pitch=0; state.zoom=.82; state.panX=0; state.panY=10; }
    else if(name==='rear'){ state.yaw=Math.PI; state.pitch=0.34; state.zoom=.80; state.panX=0; state.panY=20; }
    else if(name==='lcdRear'){ state.yaw=2.55; state.pitch=.40; state.zoom=1.05; state.panX=210; state.panY=25; }
    else if(name==='kyRear'){ state.yaw=-2.48; state.pitch=.40; state.zoom=1.05; state.panX=-210; state.panY=25; }
    else {
      // Widok montażowy / Reset: prosto od przodu i automatycznie na środku.
      state.yaw=0; state.pitch=0; state.zoom=.82; state.panX=0; state.panY=0; state.autoFrame=true;
    }
    render();
  }
  window.wiring3dView=setView;
  window.wiring3dReset=()=>setView('iso');
  window.wiring3dToggleLabels=()=>{state.labels=!state.labels;render();};

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
  canvas.addEventListener('pointerup',end); canvas.addEventListener('pointercancel',end);
  canvas.addEventListener('wheel',e=>{state.autoFrame=false;state.zoom=Math.max(.55,Math.min(2.0,state.zoom*(e.deltaY<0?1.08:.92)));render();e.preventDefault();},{passive:false});
  canvas.addEventListener('dblclick',e=>{setView('iso');e.preventDefault();});
  new ResizeObserver(render).observe(scene||canvas);
  render();
})();
