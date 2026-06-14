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
    this.fallbackPolling = false;
    this.fallbackTimer = null;
    this.fallbackInterval = 10000;

    this.handlers = {};
  }

  connect(url) {
    this.url = url;
    this.fallbackPolling = false;

    try {
      this.ws = new WebSocket(url);
    } catch (e) {
      console.warn('WS connection failed, using polling fallback:', e);
      this.startFallback();
      return;
    }

    this.ws.onopen = () => {
      this.isConnected = true;
      this.reconnectAttempts = 0;
      this.reconnectDelay = 1000;
      this.stopFallback();
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
      this.scheduleReconnect();
    };

    this.ws.onerror = () => {
      this.isConnected = false;
    };
  }

  disconnect() {
    this.stopReconnect();
    this.stopPing();
    this.stopFallback();
    if (this.ws) {
      this.ws.onclose = null;
      this.ws.close();
      this.ws = null;
    }
    this.isConnected = false;
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
      if (this.url) this.connect(this.url);
    }.bind(this), delay);
  }

  stopReconnect() {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
  }

  startFallback() {
    this.fallbackPolling = true;
  }

  stopFallback() {
    this.fallbackPolling = false;
    if (this.fallbackTimer) {
      clearInterval(this.fallbackTimer);
      this.fallbackTimer = null;
    }
  }
}
