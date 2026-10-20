'use strict';
const maskSource=location.pathname.endsWith('/mask');
let inFlight=false,lastImage='';
function draw(s){
 const p=s.preferences,score=document.getElementById('score'),mask=document.getElementById('mask');
 score.hidden=maskSource||!p.enabled;mask.hidden=!maskSource||!s.maskVisible;
 score.className=p.style+(p.backdrop?' backdrop':'');score.style.transform=`scale(${p.scale/100})`;
 for(const [id,value]of [['wins',s.wins],['losses',s.losses]])document.getElementById(id).textContent=value;
 document.getElementById('wins').style.color=p.winColor;document.getElementById('losses').style.color=p.lossColor;
 document.getElementById('win-label').hidden=!p.labels;document.getElementById('loss-label').hidden=!p.labels;
 const img=document.getElementById('mask-image'),image=s.maskImageURL||'';img.hidden=!image;document.getElementById('mask-panel').hidden=!!image;
 if(lastImage!==image){lastImage=image;if(lastImage)img.src=lastImage;else img.removeAttribute('src')}
}
async function poll(){if(inFlight)return;inFlight=true;try{const r=await fetch('state',{cache:'no-store',signal:AbortSignal.timeout(4000)});if(!r.ok)throw Error();draw(await r.json())}catch(e){if(maskSource)document.getElementById('mask').hidden=false}finally{inFlight=false}}
// The mask starts opaque before the first request and returns to opaque on disconnect.
if(maskSource)document.getElementById('mask').hidden=false;
poll();setInterval(poll,1500);
