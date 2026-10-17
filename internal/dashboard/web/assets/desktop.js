'use strict';

function initDesktopShell(){
 const runtime=window.runtime;
 if(!runtime?.WindowMinimise||!runtime?.Quit)return;
 document.documentElement.classList.add('desktop');
 const titlebar=$('window-controls');
 titlebar.hidden=false;
 titlebar.addEventListener('click',async event=>{
  const button=event.target.closest('[data-window]');
  if(!button)return;
  switch(button.dataset.window){
   case 'minimize':runtime.WindowMinimise();break;
   case 'close':runtime.Quit();break;
   case 'tray':
    try{await api('window/hide','POST')}catch(error){toast(error.message,true)}
    break;
  }
 });
}
