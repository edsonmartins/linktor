"use strict";var LinktorLoader=(()=>{function g(t,e){return`
    .linktor-widget {
      position: fixed; bottom: 20px; ${t}: 20px; z-index: ${e};
      font-family: var(--linktor-font-family, -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif);
      font-size: var(--linktor-font-size, 14px);
    }
    .linktor-launcher {
      width: var(--linktor-launcher-size, 60px); height: var(--linktor-launcher-size, 60px); border-radius: 50%;
      background-color: var(--linktor-color, #007bff); border: none; cursor: pointer;
      box-shadow: 0 4px 12px rgba(0,0,0,.15);
      display: flex; align-items: center; justify-content: center;
      transition: transform .2s, box-shadow .2s;
    }
    .linktor-launcher:hover { transform: scale(1.05); box-shadow: 0 6px 16px rgba(0,0,0,.2); }
    .linktor-launcher svg { width: 28px; height: 28px; fill: var(--linktor-launcher-icon-color, #fff); }
    .linktor-chat {
      display: none; position: absolute; bottom: 80px; ${t}: 0;
      width: var(--linktor-width, 370px); height: var(--linktor-height, 520px);
      background: var(--linktor-background, #fff); border-radius: var(--linktor-radius, 12px);
      box-shadow: 0 5px 40px rgba(0,0,0,.16); overflow: hidden; flex-direction: column;
    }
    .linktor-chat.open { display: flex; }
    .linktor-header {
      background-color: var(--linktor-header-bg, var(--linktor-color, #007bff)); color: var(--linktor-header-color, #fff); padding: 16px 20px;
      display: flex; align-items: center; justify-content: space-between;
    }
    .linktor-header-title { font-size: 16px; font-weight: 600; }
    .linktor-header-status { font-size: 12px; opacity: .9; }
    .linktor-close { background: none; border: none; color: currentColor; cursor: pointer; padding: 4px; display: flex; }
    .linktor-close svg { width: 20px; height: 20px; fill: currentColor; }
    .linktor-messages { flex: 1; overflow-y: auto; padding: 16px; display: flex; flex-direction: column; gap: 12px; }
    .linktor-message { max-width: 80%; padding: 10px 14px; border-radius: 16px; line-height: 1.4; word-wrap: break-word; }
    .linktor-message.contact { background-color: var(--linktor-agent-bubble-bg, #e9ecef); color: var(--linktor-agent-bubble-color, #333); align-self: flex-start; border-bottom-left-radius: 4px; }
    .linktor-message.user { background-color: var(--linktor-user-bubble-bg, var(--linktor-color, #007bff)); color: var(--linktor-user-bubble-color, #fff); align-self: flex-end; border-bottom-right-radius: 4px; }
    .linktor-message.system { background-color: #6c757d; color: #fff; align-self: center; font-size: 12px; }
    .linktor-message img { max-width: 100%; border-radius: 8px; display: block; }
    .linktor-message-time { font-size: 10px; opacity: .7; margin-top: 4px; }
    .linktor-typing { display: none; padding: 10px 14px; background-color: var(--linktor-agent-bubble-bg, #e9ecef); border-radius: 16px; align-self: flex-start; max-width: 60px; }
    .linktor-typing.show { display: block; }
    .linktor-typing-dots { display: flex; gap: 4px; }
    .linktor-typing-dot { width: 8px; height: 8px; background-color: #999; border-radius: 50%; animation: linktor-bounce 1.4s infinite ease-in-out; }
    .linktor-typing-dot:nth-child(1) { animation-delay: -.32s; }
    .linktor-typing-dot:nth-child(2) { animation-delay: -.16s; }
    @keyframes linktor-bounce { 0%,80%,100% { transform: scale(0); } 40% { transform: scale(1); } }
    .linktor-input-area { padding: 12px 16px; border-top: 1px solid #e9ecef; display: flex; gap: 8px; align-items: flex-end; }
    .linktor-input {
      flex: 1; border: 1px solid #e9ecef; border-radius: 20px; padding: 10px 16px; font-size: 14px;
      outline: none; resize: none; max-height: 100px; font-family: inherit;
    }
    .linktor-input:focus { border-color: var(--linktor-color, #007bff); }
    .linktor-send {
      width: 40px; height: 40px; border-radius: 50%; background-color: var(--linktor-color, #007bff);
      border: none; cursor: pointer; display: flex; align-items: center; justify-content: center; flex-shrink: 0;
    }
    .linktor-send:disabled { opacity: .5; cursor: not-allowed; }
    .linktor-send svg { width: 20px; height: 20px; fill: #fff; }
    .linktor-powered { text-align: center; padding: 8px; font-size: 11px; color: #999; border-top: 1px solid #f0f0f0; }
    .linktor-powered a { color: #666; text-decoration: none; }
    @media (max-width: 480px) {
      .linktor-chat { position: fixed; inset: 0; width: 100%; height: 100%; border-radius: 0; }
    }
  `}var f={title:"Chat",online:"Online",offline:"Offline",connecting:"Connecting\u2026",inputPlaceholder:"Type a message\u2026",sendAriaLabel:"Send",launcherAriaLabel:"Open chat",closeAriaLabel:"Close chat",poweredBy:'Powered by <a href="https://linktor.io" target="_blank" rel="noopener">Linktor</a>'},m={primaryColor:"--linktor-color",fontFamily:"--linktor-font-family",fontSize:"--linktor-font-size",width:"--linktor-width",height:"--linktor-height",borderRadius:"--linktor-radius",background:"--linktor-background",launcherSize:"--linktor-launcher-size",launcherIconColor:"--linktor-launcher-icon-color",headerBg:"--linktor-header-bg",headerColor:"--linktor-header-color",agentBubbleBg:"--linktor-agent-bubble-bg",agentBubbleColor:"--linktor-agent-bubble-color",userBubbleBg:"--linktor-user-bubble-bg",userBubbleColor:"--linktor-user-bubble-color"},k='<svg viewBox="0 0 24 24"><path d="M20 2H4c-1.1 0-2 .9-2 2v18l4-4h14c1.1 0 2-.9 2-2V4c0-1.1-.9-2-2-2zm0 14H6l-2 2V4h16v12z"/></svg>',b='<svg viewBox="0 0 24 24"><path d="M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z"/></svg>',y='<svg viewBox="0 0 24 24"><path d="M2.01 21L23 12 2.01 3 2 10l15 2-15 2z"/></svg>',x=5,u=class{constructor(t){if(this.ws=null,this.isOpen=!1,this.isConnected=!1,this.reconnectAttempts=0,this.destroyed=!1,this.listeners={},!t||!t.channelId)throw new Error("Linktor: channelId is required");if(!t.baseUrl)throw new Error("Linktor: baseUrl is required");this.cfg={position:"right",zIndex:999999,showLauncher:!0,...t},this.labels=this.resolveLabels(t),this.sessionId=this.getSessionId()}resolveLabels(t){var e;let s={...f,...t.labels||{}};return t.title&&(s.title=t.title),(e=t.labels)!=null&&e.title&&(s.title=t.labels.title),t.hidePoweredBy&&(s.poweredBy=!1),s}mount(){return this.root||this.destroyed?this:(this.injectStyles(),this.buildDom(),this.wireEvents(),this.loadConfig(),this.cfg.onReady&&this.on("ready",this.cfg.onReady),this.cfg.onOpen&&this.on("open",this.cfg.onOpen),this.cfg.onClose&&this.on("close",this.cfg.onClose),this.cfg.onMessage&&this.on("message",t=>this.cfg.onMessage(t)),this.emit("ready"),this.cfg.autoOpen&&this.open(),this)}open(){!this.chatEl||this.isOpen||(this.isOpen=!0,this.chatEl.classList.add("open"),this.ws||this.connect(),this.emit("open"))}close(){!this.chatEl||!this.isOpen||(this.isOpen=!1,this.chatEl.classList.remove("open"),this.emit("close"))}toggle(){this.isOpen?this.close():this.open()}sendMessage(t){let e=(t||"").trim();if(e){if(!this.isConnected||!this.ws){this.connect();return}this.ws.send(JSON.stringify({type:"message",payload:{id:c(),content_type:"text",content:e,timestamp:new Date().toISOString()}})),this.render({content_type:"text",content:e,sender_type:"contact",timestamp:new Date().toISOString()})}}updateUser(t){this.cfg.visitor={...this.cfg.visitor,...t}}reset(){try{sessionStorage.removeItem(this.sessionKey())}catch{}this.sessionId=this.getSessionId(),this.ws&&(this.ws.close(),this.ws=null)}destroy(){var t,e;this.destroyed=!0,this.ws&&(this.ws.onclose=null,this.ws.close(),this.ws=null),(t=this.root)==null||t.remove(),(e=this.customStyleEl)==null||e.remove(),this.listeners={}}on(t,e){var s;return((s=this.listeners)[t]||(s[t]=new Set)).add(e),()=>{var i;return(i=this.listeners[t])==null?void 0:i.delete(e)}}emit(t,e){var s;(s=this.listeners[t])==null||s.forEach(i=>{try{i(e)}catch{}})}sessionKey(){return"linktor_session_"+this.cfg.channelId}getSessionId(){try{let t=sessionStorage.getItem(this.sessionKey());return t||(t=c(),sessionStorage.setItem(this.sessionKey(),t)),t}catch{return c()}}injectStyles(){if(!document.getElementById("linktor-styles")){let t=document.createElement("style");t.id="linktor-styles",t.textContent=g(this.cfg.position,this.cfg.zIndex),document.head.appendChild(t)}if(this.cfg.customCss){let t=document.createElement("style");t.className="linktor-custom-css",t.textContent=this.cfg.customCss,document.head.appendChild(t),this.customStyleEl=t}}buildDom(){let t=this.labels,e=this.cfg.launcherIcon||k,s=t.poweredBy?`<div class="linktor-powered">${t.poweredBy}</div>`:"",i=document.createElement("div");i.className="linktor-widget",i.innerHTML=`
      <div class="linktor-chat" role="dialog" aria-label="${l(t.title)}">
        <div class="linktor-header">
          <div>
            <div class="linktor-header-title"></div>
            <div class="linktor-header-status"></div>
          </div>
          <button class="linktor-close" aria-label="${l(t.closeAriaLabel)}">${b}</button>
        </div>
        <div class="linktor-messages">
          <div class="linktor-typing"><div class="linktor-typing-dots">
            <div class="linktor-typing-dot"></div><div class="linktor-typing-dot"></div><div class="linktor-typing-dot"></div>
          </div></div>
        </div>
        <div class="linktor-input-area">
          <textarea class="linktor-input" placeholder="${l(t.inputPlaceholder)}" rows="1"></textarea>
          <button class="linktor-send" aria-label="${l(t.sendAriaLabel)}" disabled>${y}</button>
        </div>
        ${s}
      </div>
      ${this.cfg.showLauncher?`<button class="linktor-launcher" aria-label="${l(t.launcherAriaLabel)}">${e}</button>`:""}
    `,document.body.appendChild(i),this.root=i,this.chatEl=i.querySelector(".linktor-chat"),this.messagesEl=i.querySelector(".linktor-messages"),this.typingEl=i.querySelector(".linktor-typing"),this.inputEl=i.querySelector(".linktor-input"),this.sendEl=i.querySelector(".linktor-send"),this.titleEl=i.querySelector(".linktor-header-title"),this.statusEl=i.querySelector(".linktor-header-status"),this.applyTheme(),this.titleEl.textContent=t.title,this.setStatus("connecting")}applyTheme(t){if(!this.root)return;let e={...this.cfg.theme,...t};this.cfg.primaryColor&&e.primaryColor===void 0&&(e.primaryColor=this.cfg.primaryColor);for(let s of Object.keys(e)){let i=e[s];i&&this.root.style.setProperty(m[s],i)}}wireEvents(){var t,e,s,i,n,r,a;(e=(t=this.root)==null?void 0:t.querySelector(".linktor-launcher"))==null||e.addEventListener("click",()=>this.toggle()),(i=(s=this.root)==null?void 0:s.querySelector(".linktor-close"))==null||i.addEventListener("click",()=>this.close()),(n=this.inputEl)==null||n.addEventListener("input",()=>{this.sendEl.disabled=!this.inputEl.value.trim(),this.autoResize()}),(r=this.inputEl)==null||r.addEventListener("keydown",d=>{d.key==="Enter"&&!d.shiftKey&&(d.preventDefault(),this.flushInput())}),(a=this.sendEl)==null||a.addEventListener("click",()=>this.flushInput())}flushInput(){let t=this.inputEl.value.trim();t&&(this.sendMessage(t),this.inputEl.value="",this.inputEl.style.height="auto",this.sendEl.disabled=!0)}autoResize(){let t=this.inputEl;t.style.height="auto",t.style.height=Math.min(t.scrollHeight,100)+"px"}async loadConfig(){var t,e;try{let s=await fetch(`${this.cfg.baseUrl}/api/v1/webchat/${this.cfg.channelId}/config`);if(!s.ok)return;let i=await s.json();!this.cfg.primaryColor&&!((t=this.cfg.theme)!=null&&t.primaryColor)&&i.widget_color&&this.applyTheme({primaryColor:i.widget_color}),!(this.cfg.title||(e=this.cfg.labels)!=null&&e.title)&&i.widget_title&&this.titleEl&&(this.titleEl.textContent=i.widget_title)}catch{}}connect(){var t,e,s;if(this.destroyed)return;let i=typeof location!="undefined"&&location.protocol==="https:"?"wss:":"ws:",n=this.cfg.baseUrl.replace(/^https?:/,i),r=new URLSearchParams({session_id:this.sessionId});(t=this.cfg.visitor)!=null&&t.name&&r.set("name",this.cfg.visitor.name),(e=this.cfg.visitor)!=null&&e.email&&r.set("email",this.cfg.visitor.email),(s=this.cfg.visitor)!=null&&s.phone&&r.set("phone",this.cfg.visitor.phone),this.cfg.token&&r.set("token",this.cfg.token),this.ws=new WebSocket(`${n}/ws/${this.cfg.channelId}?${r.toString()}`),this.ws.onopen=()=>{this.isConnected=!0,this.reconnectAttempts=0,this.setStatus("online"),this.emit("connected")},this.ws.onmessage=a=>{try{this.handleFrame(JSON.parse(a.data))}catch{}},this.ws.onclose=()=>{this.isConnected=!1,this.setStatus("offline"),this.emit("disconnected"),!this.destroyed&&this.reconnectAttempts<x&&(this.reconnectAttempts++,setTimeout(()=>this.connect(),Math.min(1e3*2**this.reconnectAttempts,3e4)))},this.ws.onerror=()=>{}}handleFrame(t){var e,s,i;switch(t.type){case"connect":if((s=(e=t.payload)==null?void 0:e.metadata)!=null&&s.session_id){this.sessionId=t.payload.metadata.session_id;try{sessionStorage.setItem(this.sessionKey(),this.sessionId)}catch{}}break;case"message":this.render(t.payload),this.emit("message",t.payload);break;case"typing":this.showTyping(!!((i=t.payload)!=null&&i.is_typing));break}}render(t){if(!this.messagesEl||!this.typingEl)return;let e=this.buildMessageEl(t),s=e;if(this.cfg.renderMessage){let i=this.cfg.renderMessage(t,{defaultElement:e,formatTime:p});if(i instanceof HTMLElement)s=i;else if(typeof i=="string"){let n=document.createElement("div");n.className="linktor-message "+(t.sender_type||"contact"),n.innerHTML=i,s=n}}this.messagesEl.insertBefore(s,this.typingEl),this.messagesEl.scrollTop=this.messagesEl.scrollHeight}buildMessageEl(t){var e;let s=document.createElement("div");s.className="linktor-message "+(t.sender_type||"contact");let i=(e=t.attachments)==null?void 0:e[0];if((i==null?void 0:i.type)==="image"&&i.url){let r=document.createElement("img");r.src=i.url,r.alt=i.filename||"image",s.appendChild(r)}if(t.content){let r=document.createElement("div");r.textContent=t.content,s.appendChild(r)}let n=document.createElement("div");return n.className="linktor-message-time",n.textContent=p(t.timestamp),s.appendChild(n),s}showTyping(t){var e;(e=this.typingEl)==null||e.classList.toggle("show",t),t&&this.messagesEl&&(this.messagesEl.scrollTop=this.messagesEl.scrollHeight)}setStatus(t){this.statusEl&&(this.statusEl.textContent=this.labels[t])}};function l(t){return t.replace(/&/g,"&amp;").replace(/"/g,"&quot;").replace(/</g,"&lt;").replace(/>/g,"&gt;")}function p(t){return(t?new Date(t):new Date).toLocaleTimeString([],{hour:"2-digit",minute:"2-digit"})}function c(){return typeof crypto!="undefined"&&"randomUUID"in crypto?crypto.randomUUID():"xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx".replace(/[xy]/g,t=>{let e=Math.random()*16|0;return(t==="x"?e:e&3|8).toString(16)})}var o=null;function h(t,e){switch(t){case"init":o||(o=new u(e).mount());break;case"open":o==null||o.open();break;case"close":o==null||o.close();break;case"toggle":o==null||o.toggle();break;case"sendMessage":o==null||o.sendMessage(typeof e=="string"?e:(e==null?void 0:e.text)||"");break;case"updateUser":o==null||o.updateUser(e);break;case"reset":o==null||o.reset();break;case"destroy":o==null||o.destroy(),o=null;break;default:console.warn("Linktor: unknown command",t)}}function v(){let t=window,e=t.LinktorObject||"linktor",s=t[e],i=s&&s.q?Array.from(s.q):[];t[e]=(n,r)=>h(n,r),i.forEach(n=>h(n[0],n[1])),!o&&t.LinktorSettings&&t.LinktorSettings.channelId&&h("init",t.LinktorSettings)}v();})();
//# sourceMappingURL=linktor.js.map