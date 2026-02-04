/**
 * Linktor Web Chat Widget
 * Embeddable chat widget for websites
 *
 * Usage:
 * <script>
 *   window.LinktorSettings = {
 *     channelId: 'your-channel-id',
 *     baseUrl: 'https://your-linktor-instance.com'
 *   };
 * </script>
 * <script src="https://your-linktor-instance.com/widget.js" async></script>
 */

(function() {
  'use strict';

  // Configuration
  const settings = window.LinktorSettings || {};
  const channelId = settings.channelId;
  const baseUrl = settings.baseUrl || '';
  const position = settings.position || 'right'; // 'left' or 'right'
  const zIndex = settings.zIndex || 999999;

  if (!channelId) {
    console.error('Linktor: channelId is required');
    return;
  }

  // State
  let ws = null;
  let sessionId = null;
  let config = {};
  let messages = [];
  let isOpen = false;
  let isConnected = false;
  let reconnectAttempts = 0;
  const maxReconnectAttempts = 5;

  // Session storage key
  const SESSION_KEY = 'linktor_session_' + channelId;

  // Get or create session ID
  function getSessionId() {
    let id = sessionStorage.getItem(SESSION_KEY);
    if (!id) {
      id = generateUUID();
      sessionStorage.setItem(SESSION_KEY, id);
    }
    return id;
  }

  // Generate UUID
  function generateUUID() {
    return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, function(c) {
      const r = Math.random() * 16 | 0;
      const v = c === 'x' ? r : (r & 0x3 | 0x8);
      return v.toString(16);
    });
  }

  // Create styles
  function createStyles() {
    const style = document.createElement('style');
    style.textContent = `
      .linktor-widget {
        position: fixed;
        bottom: 20px;
        ${position}: 20px;
        z-index: ${zIndex};
        font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
        font-size: 14px;
      }

      .linktor-launcher {
        width: 60px;
        height: 60px;
        border-radius: 50%;
        background-color: var(--linktor-color, #007bff);
        border: none;
        cursor: pointer;
        box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
        display: flex;
        align-items: center;
        justify-content: center;
        transition: transform 0.2s, box-shadow 0.2s;
      }

      .linktor-launcher:hover {
        transform: scale(1.05);
        box-shadow: 0 6px 16px rgba(0, 0, 0, 0.2);
      }

      .linktor-launcher svg {
        width: 28px;
        height: 28px;
        fill: white;
      }

      .linktor-chat {
        display: none;
        position: absolute;
        bottom: 80px;
        ${position}: 0;
        width: 370px;
        height: 520px;
        background: white;
        border-radius: 12px;
        box-shadow: 0 5px 40px rgba(0, 0, 0, 0.16);
        overflow: hidden;
        flex-direction: column;
      }

      .linktor-chat.open {
        display: flex;
      }

      .linktor-header {
        background-color: var(--linktor-color, #007bff);
        color: white;
        padding: 16px 20px;
        display: flex;
        align-items: center;
        justify-content: space-between;
      }

      .linktor-header-title {
        font-size: 16px;
        font-weight: 600;
      }

      .linktor-header-status {
        font-size: 12px;
        opacity: 0.9;
      }

      .linktor-close {
        background: none;
        border: none;
        color: white;
        cursor: pointer;
        padding: 4px;
        display: flex;
        align-items: center;
        justify-content: center;
      }

      .linktor-close svg {
        width: 20px;
        height: 20px;
        fill: currentColor;
      }

      .linktor-messages {
        flex: 1;
        overflow-y: auto;
        padding: 16px;
        display: flex;
        flex-direction: column;
        gap: 12px;
      }

      .linktor-message {
        max-width: 80%;
        padding: 10px 14px;
        border-radius: 16px;
        line-height: 1.4;
        word-wrap: break-word;
      }

      .linktor-message.contact {
        background-color: #e9ecef;
        color: #333;
        align-self: flex-start;
        border-bottom-left-radius: 4px;
      }

      .linktor-message.user,
      .linktor-message.system {
        background-color: var(--linktor-color, #007bff);
        color: white;
        align-self: flex-end;
        border-bottom-right-radius: 4px;
      }

      .linktor-message.system {
        background-color: #6c757d;
        align-self: center;
        font-size: 12px;
      }

      .linktor-message-time {
        font-size: 10px;
        opacity: 0.7;
        margin-top: 4px;
      }

      .linktor-typing {
        display: none;
        padding: 10px 14px;
        background-color: #e9ecef;
        border-radius: 16px;
        align-self: flex-start;
        max-width: 60px;
      }

      .linktor-typing.show {
        display: block;
      }

      .linktor-typing-dots {
        display: flex;
        gap: 4px;
      }

      .linktor-typing-dot {
        width: 8px;
        height: 8px;
        background-color: #999;
        border-radius: 50%;
        animation: linktor-bounce 1.4s infinite ease-in-out;
      }

      .linktor-typing-dot:nth-child(1) { animation-delay: -0.32s; }
      .linktor-typing-dot:nth-child(2) { animation-delay: -0.16s; }

      @keyframes linktor-bounce {
        0%, 80%, 100% { transform: scale(0); }
        40% { transform: scale(1); }
      }

      .linktor-input-area {
        padding: 12px 16px;
        border-top: 1px solid #e9ecef;
        display: flex;
        gap: 8px;
        align-items: flex-end;
      }

      .linktor-input {
        flex: 1;
        border: 1px solid #e9ecef;
        border-radius: 20px;
        padding: 10px 16px;
        font-size: 14px;
        outline: none;
        resize: none;
        max-height: 100px;
        font-family: inherit;
      }

      .linktor-input:focus {
        border-color: var(--linktor-color, #007bff);
      }

      .linktor-send {
        width: 40px;
        height: 40px;
        border-radius: 50%;
        background-color: var(--linktor-color, #007bff);
        border: none;
        cursor: pointer;
        display: flex;
        align-items: center;
        justify-content: center;
        flex-shrink: 0;
      }

      .linktor-send:disabled {
        opacity: 0.5;
        cursor: not-allowed;
      }

      .linktor-send svg {
        width: 20px;
        height: 20px;
        fill: white;
      }

      .linktor-powered {
        text-align: center;
        padding: 8px;
        font-size: 11px;
        color: #999;
        border-top: 1px solid #f0f0f0;
      }

      .linktor-powered a {
        color: #666;
        text-decoration: none;
      }

      @media (max-width: 480px) {
        .linktor-chat {
          position: fixed;
          top: 0;
          left: 0;
          right: 0;
          bottom: 0;
          width: 100%;
          height: 100%;
          border-radius: 0;
        }
      }
    `;
    document.head.appendChild(style);
  }

  // Create widget HTML
  function createWidget() {
    const widget = document.createElement('div');
    widget.className = 'linktor-widget';
    widget.innerHTML = `
      <div class="linktor-chat" id="linktor-chat">
        <div class="linktor-header">
          <div>
            <div class="linktor-header-title" id="linktor-title">Chat</div>
            <div class="linktor-header-status" id="linktor-status">Connecting...</div>
          </div>
          <button class="linktor-close" id="linktor-close">
            <svg viewBox="0 0 24 24"><path d="M19 6.41L17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z"/></svg>
          </button>
        </div>
        <div class="linktor-messages" id="linktor-messages">
          <div class="linktor-typing" id="linktor-typing">
            <div class="linktor-typing-dots">
              <div class="linktor-typing-dot"></div>
              <div class="linktor-typing-dot"></div>
              <div class="linktor-typing-dot"></div>
            </div>
          </div>
        </div>
        <div class="linktor-input-area">
          <textarea
            class="linktor-input"
            id="linktor-input"
            placeholder="Type a message..."
            rows="1"
          ></textarea>
          <button class="linktor-send" id="linktor-send" disabled>
            <svg viewBox="0 0 24 24"><path d="M2.01 21L23 12 2.01 3 2 10l15 2-15 2z"/></svg>
          </button>
        </div>
        <div class="linktor-powered">
          Powered by <a href="https://linktor.io" target="_blank">Linktor</a>
        </div>
      </div>
      <button class="linktor-launcher" id="linktor-launcher">
        <svg viewBox="0 0 24 24"><path d="M20 2H4c-1.1 0-2 .9-2 2v18l4-4h14c1.1 0 2-.9 2-2V4c0-1.1-.9-2-2-2zm0 14H6l-2 2V4h16v12z"/></svg>
      </button>
    `;
    document.body.appendChild(widget);
    return widget;
  }

  // Initialize widget
  function init() {
    createStyles();
    const widget = createWidget();
    sessionId = getSessionId();

    // DOM elements
    const launcher = document.getElementById('linktor-launcher');
    const chat = document.getElementById('linktor-chat');
    const closeBtn = document.getElementById('linktor-close');
    const input = document.getElementById('linktor-input');
    const sendBtn = document.getElementById('linktor-send');
    const messagesContainer = document.getElementById('linktor-messages');
    const titleEl = document.getElementById('linktor-title');
    const statusEl = document.getElementById('linktor-status');
    const typingEl = document.getElementById('linktor-typing');

    // Load config
    loadConfig().then(cfg => {
      config = cfg;
      document.documentElement.style.setProperty('--linktor-color', config.widget_color || '#007bff');
      titleEl.textContent = config.widget_title || 'Chat';
    });

    // Event handlers
    launcher.addEventListener('click', () => {
      isOpen = !isOpen;
      chat.classList.toggle('open', isOpen);
      if (isOpen && !ws) {
        connect();
      }
    });

    closeBtn.addEventListener('click', () => {
      isOpen = false;
      chat.classList.remove('open');
    });

    input.addEventListener('input', () => {
      sendBtn.disabled = !input.value.trim();
      autoResize(input);
    });

    input.addEventListener('keydown', (e) => {
      if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault();
        sendMessage();
      }
    });

    sendBtn.addEventListener('click', sendMessage);

    // Auto-resize textarea
    function autoResize(el) {
      el.style.height = 'auto';
      el.style.height = Math.min(el.scrollHeight, 100) + 'px';
    }

    // Send message
    function sendMessage() {
      const content = input.value.trim();
      if (!content || !isConnected) return;

      const msg = {
        type: 'message',
        payload: {
          id: generateUUID(),
          content_type: 'text',
          content: content,
          timestamp: new Date().toISOString()
        }
      };

      ws.send(JSON.stringify(msg));
      addMessage({
        content_type: 'text',
        content: content,
        sender_type: 'contact',
        timestamp: new Date().toISOString()
      });

      input.value = '';
      input.style.height = 'auto';
      sendBtn.disabled = true;
    }

    // Add message to UI
    function addMessage(msg) {
      const messageEl = document.createElement('div');
      messageEl.className = 'linktor-message ' + (msg.sender_type || 'contact');

      const content = document.createElement('div');
      content.textContent = msg.content;
      messageEl.appendChild(content);

      const time = document.createElement('div');
      time.className = 'linktor-message-time';
      time.textContent = formatTime(msg.timestamp);
      messageEl.appendChild(time);

      messagesContainer.insertBefore(messageEl, typingEl);
      messagesContainer.scrollTop = messagesContainer.scrollHeight;
      messages.push(msg);
    }

    // Format time
    function formatTime(timestamp) {
      const date = new Date(timestamp);
      return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    }

    // Show/hide typing indicator
    function showTyping(show) {
      typingEl.classList.toggle('show', show);
      if (show) {
        messagesContainer.scrollTop = messagesContainer.scrollHeight;
      }
    }

    // Update status
    function updateStatus(status) {
      statusEl.textContent = status;
    }

    // Connect to WebSocket
    function connect() {
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
      const wsUrl = baseUrl.replace(/^https?:/, protocol) + '/ws/' + channelId + '?session_id=' + sessionId;

      // Add visitor info if available
      const params = new URLSearchParams();
      if (settings.visitorName) params.append('name', settings.visitorName);
      if (settings.visitorEmail) params.append('email', settings.visitorEmail);

      ws = new WebSocket(wsUrl + (params.toString() ? '&' + params.toString() : ''));

      ws.onopen = () => {
        isConnected = true;
        reconnectAttempts = 0;
        updateStatus('Online');
      };

      ws.onmessage = (event) => {
        try {
          const msg = JSON.parse(event.data);
          handleMessage(msg);
        } catch (e) {
          console.error('Linktor: Failed to parse message', e);
        }
      };

      ws.onclose = () => {
        isConnected = false;
        updateStatus('Offline');

        // Reconnect logic
        if (isOpen && reconnectAttempts < maxReconnectAttempts) {
          reconnectAttempts++;
          setTimeout(connect, Math.min(1000 * Math.pow(2, reconnectAttempts), 30000));
        }
      };

      ws.onerror = (error) => {
        console.error('Linktor: WebSocket error', error);
      };
    }

    // Handle incoming messages
    function handleMessage(msg) {
      switch (msg.type) {
        case 'connect':
          if (msg.payload.metadata) {
            sessionId = msg.payload.metadata.session_id || sessionId;
            sessionStorage.setItem(SESSION_KEY, sessionId);
          }
          break;

        case 'message':
          addMessage(msg.payload);
          break;

        case 'typing':
          showTyping(msg.payload.is_typing);
          break;

        case 'read':
          // Mark message as read
          break;

        case 'ack':
          // Message acknowledged
          break;

        case 'error':
          console.error('Linktor: Server error', msg.payload.error);
          break;
      }
    }
  }

  // Load widget config from server
  async function loadConfig() {
    try {
      const response = await fetch(baseUrl + '/api/v1/webchat/' + channelId + '/config');
      if (response.ok) {
        return await response.json();
      }
    } catch (e) {
      console.error('Linktor: Failed to load config', e);
    }
    return {};
  }

  // Initialize when DOM is ready
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();
