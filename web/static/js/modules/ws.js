class WsClient {
  constructor() {
    this.ws = null;
    this.url = null;
    this.reconnectAttempts = 0;
    this.maxReconnectDelay = 30000;
    this.reconnectDelay = 1000;
    this.reconnectTimer = null;
    this.pingTimer = null;
    this.isConnected = false;

    this.handlers = {};
    this._indicator = null;
    this._ensureIndicator();
    this._updateIndicator('disconnected');
  }

  connect(url) {
    this.url = url;
    this._updateIndicator('connecting');

    try {
      this.ws = new WebSocket(url);
    } catch (e) {
      console.warn('WS connection failed:', e);
      this._updateIndicator('disconnected');
      return;
    }

    this.ws.onopen = () => {
      this.isConnected = true;
      this.reconnectAttempts = 0;
      this.reconnectDelay = 1000;
      this._updateIndicator('connected');
      this.startPing();
    };

    this.ws.onmessage = (event) => {
      try {
        var frame = JSON.parse(event.data);
        this.handleFrame(frame);
      } catch (e) {
        console.error('WS parse error:', e);
      }
    };

    this.ws.onclose = () => {
      this.isConnected = false;
      this.stopPing();
      this._updateIndicator('reconnecting');
      this.scheduleReconnect();
    };

    this.ws.onerror = () => {
      this.isConnected = false;
    };
  }

  disconnect() {
    this.stopReconnect();
    this.stopPing();
    if (this.ws) {
      this.ws.onclose = null;
      this.ws.close();
      this.ws = null;
    }
    this.isConnected = false;
    this._updateIndicator('disconnected');
  }

  send(data) {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(data));
      return true;
    }
    return false;
  }

  on(event, callback) {
    this.handlers[event] = this.handlers[event] || [];
    this.handlers[event].push(callback);
  }

  off(event, callback) {
    var list = this.handlers[event];
    if (!list) return;
    this.handlers[event] = list.filter(function(fn) { return fn !== callback; });
  }

  handleFrame(frame) {
    var type = frame.event_type;
    var list = this.handlers[type];
    if (list) {
      for (var i = 0; i < list.length; i++) {
        list[i](frame);
      }
    }
    var wildcard = this.handlers['*'];
    if (wildcard) {
      for (var i = 0; i < wildcard.length; i++) {
        wildcard[i](frame);
      }
    }
  }

  sendTypingStart(conversationId, userId, userType, clientId, businessId) {
    this.send({
      event_type: 3,
      conversation_id: String(conversationId),
      sender_id: String(userId),
      sender_type: userType,
      timestamp: Date.now(),
      typing: {
        user_id: String(userId),
        user_type: userType,
        conversation_id: String(conversationId),
        client_id: String(clientId),
        business_id: String(businessId)
      }
    });
  }

  sendTypingStop(conversationId, userId, userType, clientId, businessId) {
    this.send({
      event_type: 4,
      conversation_id: String(conversationId),
      sender_id: String(userId),
      sender_type: userType,
      timestamp: Date.now(),
      typing: {
        user_id: String(userId),
        user_type: userType,
        conversation_id: String(conversationId),
        client_id: String(clientId),
        business_id: String(businessId)
      }
    });
  }

  sendDeliveredAck(conversationId, clientId) {
    this.send({
      event_type: 11,
      conversation_id: String(conversationId),
      sender_type: 'client',
      timestamp: Date.now(),
      delivered_ack: {
        conversation_id: String(conversationId),
        client_id: String(clientId)
      }
    });
  }

  startPing() {
    this.pingTimer = setInterval(function() {
      this.send({ event_type: 9 });
    }.bind(this), 30000);
  }

  stopPing() {
    if (this.pingTimer) {
      clearInterval(this.pingTimer);
      this.pingTimer = null;
    }
  }

  scheduleReconnect() {
    if (this.reconnectTimer) return;
    var delay = Math.min(this.reconnectDelay, this.maxReconnectDelay);
    this.reconnectTimer = setTimeout(function() {
      this.reconnectTimer = null;
      this.reconnectAttempts++;
      this.reconnectDelay *= 2;
      this._updateIndicator('connecting');
      if (this.url) this.connect(this.url);
    }.bind(this), delay);
  }

  stopReconnect() {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
  }

  _ensureIndicator() {
    if (this._indicator) return;
    var dot = document.createElement('div');
    dot.id = 'wsIndicator';
    dot.style.cssText = 'position:fixed;bottom:12px;right:12px;width:10px;height:10px;border-radius:50%;z-index:9999;border:2px solid var(--color-surface,#fff);box-shadow:0 1px 3px rgba(0,0,0,0.3);transition:background .3s;cursor:pointer';
    dot.title = 'WebSocket: disconnected';
    dot.addEventListener('click', function() {
      var s = ['disconnected','connecting','connected','reconnecting'];
      var labels = ['Disconnected','Connecting...','Connected','Reconnecting...'];
      var states = ['#f43f5e','#f59e0b','#10b981','#f59e0b'];
      var idx = s.indexOf(dot.getAttribute('data-ws-state'));
      if (idx >= 0 && typeof showNotification !== 'undefined') {
        showNotification('WebSocket: ' + labels[idx], idx === 2 ? 'success' : 'warning');
      }
    });
    document.body.appendChild(dot);
    this._indicator = dot;
  }

  _updateIndicator(state) {
    this._ensureIndicator();
    var dot = this._indicator;
    dot.setAttribute('data-ws-state', state);
    var colors = { connected: '#10b981', connecting: '#f59e0b', reconnecting: '#f59e0b', disconnected: '#f43f5e' };
    var labels = { connected: 'Connected', connecting: 'Connecting...', reconnecting: 'Reconnecting...', disconnected: 'Disconnected' };
    dot.style.background = colors[state] || '#94a3b8';
    dot.title = 'WebSocket: ' + (labels[state] || state);
  }
}
