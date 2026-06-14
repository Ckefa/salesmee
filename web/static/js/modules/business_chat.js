if (window.wsClient) window.wsClient.disconnect();
var wsClient = null;
if (window.typingTimeout) clearTimeout(window.typingTimeout);
var typingTimeout = null;

scrollToBottom();
markAsRead();
startWsClient();

function scrollToBottom() {
  var container = document.getElementById('messages-container');
  if (container) requestAnimationFrame(function() {
    container.scrollTop = container.scrollHeight;
  });
}

function markAsRead() {
  fetch(`/business/clients/${clientId}/read`, { method: 'PUT', headers: { 'X-CSRF-Token': getCookie('csrf_token') } })
    .then(function() {
      var badge = document.querySelector('.client-item[data-client-id="' + clientId + '"] .unread-badge');
      if (badge) badge.remove();
    })
    .catch(console.error);
}

function tickSvg(state) {
  if (state === 'read') {
    return '<svg viewBox="0 0 16 12" width="14" height="11" fill="none" stroke="#53bdeb" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M2 6L5 9L11 3"/><path d="M6 6L9 9L15 3"/></svg>';
  }
  if (state === 'delivered') {
    return '<svg viewBox="0 0 16 12" width="14" height="11" fill="none" stroke="#8696a0" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M2 6L5 9L11 3"/><path d="M6 6L9 9L15 3"/></svg>';
  }
  return '<svg viewBox="0 0 12 12" width="12" height="11" fill="none" stroke="#8696a0" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M2 6L5 9L11 3"/></svg>';
}

function setMessageTickState(item, state) {
  if (!item) return;
  var tick = item.querySelector('.msg-tick');
  if (!tick) return;
  if (tick.getAttribute('data-read-state') === 'read') return;
  tick.setAttribute('data-read-state', state);
  tick.innerHTML = tickSvg(state);
  tick.style.width = state === 'sent' ? '12px' : '14px';
}

function applyReadReceipt(receipt) {
  if (!receipt || receipt.reader_type === 'business') return;
  if (receipt.conversation_id && receipt.conversation_id !== String(conversationId)) return;

  if (receipt.message_id) {
    setMessageTickState(document.querySelector('#messages-container .message-item.out[data-message-id="' + receipt.message_id + '"]'), 'read');
    return;
  }

  document.querySelectorAll('#messages-container .message-item.out').forEach(function(item) {
    setMessageTickState(item, 'read');
  });
}

function markVisibleConversationRead() {
  if (document.visibilityState === 'hidden') return;
  markAsRead();
}

function reloadBusinessChatFromServer() {
  if (!clientId) return;
  fetch('clients/' + clientId + '/messages')
    .then(function(r) { return r.text(); })
    .then(function(html) {
      var parser = new DOMParser();
      var doc = parser.parseFromString(html, 'text/html');
      var next = doc.getElementById('messages-container');
      var current = document.getElementById('messages-container');
      if (next && current) {
        current.innerHTML = next.innerHTML;
        scrollToBottom();
      }
    })
    .catch(console.error);
}

function wsToken() {
  return getCookie('token') || getCookie('team_token') || '';
}

function startWsClient() {
  if (wsClient) wsClient.disconnect();
  wsClient = new WsClient();
  var token = wsToken();
  if (!token) return;
  wsClient.connect('/ws/business?token=' + encodeURIComponent(token) + '&business_id=' + businessId);

  wsClient.on(1, function(frame) {
    if (frame.conversation_id && frame.conversation_id !== String(conversationId)) return;
    var container = document.getElementById('messages-container');
    if (!container) return;
    var msg = frame.new_message;
    if (!msg) return;
    if (msg.msg_type === 'order') {
      reloadBusinessChatFromServer();
      return;
    } else if (msg.msg_type === 'booking') {
      reloadBusinessChatFromServer();
      return;
    }
    if (frame.sender_type === 'business') return;
    var html = '';
    if (msg.media_url) {
      html = renderMediaMessage(msg);
    } else {
      html = '<div class="msg in message-item" data-message-id="' + msg.id + '"><div class="msg-bbl"><svg class="msg-tail" viewBox="0 0 10 15" height="15" width="10" preserveAspectRatio="xMidYMid meet"><path fill="var(--color-bg)" d="M1,3L10,14V1H3C1.5,1,0.5,2,1,3z"></path><path fill="currentColor" d="M1,2L10,13V0H3C1.5,0,0.5,1,1,2z"></path></svg><span class="msg-txt">' + escapeHtml(msg.content || '') + '</span><span class="msg-meta"><span class="msg-time">' + formatTime(msg.created_at) + '</span></span></div></div>';
    }
    container.insertAdjacentHTML('beforeend', html);
    scrollToBottom();
    markVisibleConversationRead();
    playNotificationSound();
  });

  wsClient.on(3, function(frame) {
    showTypingIndicator(frame.typing);
  });
  wsClient.on(4, function(frame) {
    hideTypingIndicator(frame.typing);
  });

  wsClient.on(6, function(frame) {
    var upd = frame.order_update;
    if (!upd) return;
    reloadBusinessChatFromServer();
  });

  wsClient.on(7, function(frame) {
    var upd = frame.booking_update;
    if (!upd) return;
    reloadBusinessChatFromServer();
  });

  wsClient.on(8, function(frame) {
    var badge = document.querySelector('.wa-unread-badge');
    if (badge && frame.unread_count) {
      badge.textContent = frame.unread_count.count;
    }
  });

  wsClient.on(2, function(frame) {
    applyReadReceipt(frame.read_receipt);
  });
}

document.addEventListener('visibilitychange', function() {
  if (document.visibilityState === 'visible') markAsRead();
});

function renderMediaMessage(msg) {
  var url = '/static/' + escapeHtml(msg.media_url);
  var mediaTag = '';
  if (msg.media_type === 'image') {
    mediaTag = '<img src="' + url + '" alt="Image" class="wa-media-image" onclick="window.open(this.src)" loading="lazy">';
  } else if (msg.media_type === 'document') {
    mediaTag = '<div class="wa-media-doc"><i class="fas fa-file-alt wa-media-doc-icon"></i><a href="' + url + '" target="_blank" class="wa-media-doc-link">' + escapeHtml(msg.media_url.split('/').pop()) + '</a><i class="fas fa-external-link-alt wa-media-doc-ext"></i></div>';
  } else if (msg.media_type === 'audio') {
    mediaTag = '<div class="wa-media-audio"><audio controls class="wa-audio-player" preload="metadata"><source src="' + url + '"></audio></div>';
  } else {
    mediaTag = '<a href="' + url + '" target="_blank" class="wa-media-doc-link"><i class="fas fa-file"></i> ' + escapeHtml(msg.media_url.split('/').pop()) + '</a>';
  }
  var inner = mediaTag + (msg.content ? '<p>' + escapeHtml(msg.content) + '</p>' : '') + '<span class="msg-meta"><span class="msg-time">' + formatTime(msg.created_at) + '</span></span>';
  return '<div class="msg in message-item" data-message-id="' + msg.id + '"><div class="msg-bbl" style="padding:3px;"><svg class="msg-tail" viewBox="0 0 10 15" height="15" width="10" preserveAspectRatio="xMidYMid meet"><path fill="var(--color-bg)" d="M1,3L10,14V1H3C1.5,1,0.5,2,1,3z"></path><path fill="currentColor" d="M1,2L10,13V0H3C1.5,0,0.5,1,1,2z"></path></svg>' + inner + '</div></div>';
}

function escapeHtml(str) {
  if (!str) return '';
  var div = document.createElement('div');
  div.appendChild(document.createTextNode(str));
  return div.innerHTML;
}

function formatTime(ts) {
  if (!ts) return '';
  var d = new Date(Number(ts));
  return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
}

function showTypingIndicator(typing) {
  if (!typing || typing.conversation_id !== String(conversationId)) return;
  if (typing.user_type === 'business') return;
  var el = document.getElementById('typingIndicator');
  if (!el) {
    el = document.createElement('div');
    el.id = 'typingIndicator';
    el.className = 'msg in';
    el.innerHTML = '<div class="msg-bbl typing"><span class="typing-label">typing</span><span class="typing-dot"></span><span class="typing-dot"></span><span class="typing-dot"></span></div>';
    document.getElementById('messages-container').appendChild(el);
  }
  scrollToBottom();
}

function hideTypingIndicator(typing) {
  if (!typing || typing.conversation_id !== String(conversationId)) return;
  if (typing.user_type === 'business') return;
  var el = document.getElementById('typingIndicator');
  if (el) el.remove();
}

function playNotificationSound() {
  try {
    var audio = new Audio('/static/sounds/notification.mp3');
    audio.volume = 0.3;
    audio.play();
  } catch(e) {}
}

// ========== Context Menu ==========

var contextMenuEl = null;
var contextMessageId = null;

function ensureContextMenu() {
  if (!contextMenuEl || !document.body.contains(contextMenuEl)) {
    contextMenuEl = document.createElement('div');
    contextMenuEl.id = 'contextMenu';
    contextMenuEl.style.cssText = 'display:none;position:fixed;z-index:9999;width:190px;border-radius:12px;border:1px solid var(--color-border);background:var(--color-surface);box-shadow:0 10px 40px rgba(0,0,0,0.15);padding:4px 0;font-size:13px;';
    contextMenuEl.innerHTML =
      '<button onclick="markMessageRead()" style="width:100%;padding:10px 16px;text-align:left;color:var(--color-info);background:none;border:none;cursor:pointer;display:flex;align-items:center;gap:8px;font-weight:500;font-size:13px;border-bottom:1px solid var(--color-border);"><svg style="width:14px;height:14px;flex-shrink:0" viewBox="0 0 512 512" fill="currentColor"><path d="M256 512c141.4 0 256-114.6 256-256S397.4 0 256 0S0 114.6 0 256S114.6 512 256 512zM369 209L241 337c-9.4 9.4-24.6 9.4-33.9 0l-64-64c-9.4-9.4-9.4-24.6 0-33.9s24.6-9.4 33.9 0l47 47L335 175c9.4-9.4 24.6-9.4 33.9 0s9.4 24.6 0 33.9z"/></svg> Mark as Read</button>' +
      '<button onclick="deleteContextMenuItem()" style="width:100%;padding:10px 16px;text-align:left;color:var(--color-error);background:none;border:none;cursor:pointer;display:flex;align-items:center;gap:8px;font-weight:500;font-size:13px;"><svg style="width:14px;height:14px;flex-shrink:0" viewBox="0 0 448 512" fill="currentColor"><path d="M135.2 17.7L128 32H32C14.3 32 0 46.3 0 64s14.3 32 32 32h384c17.7 0 32-14.3 32-32s-14.3-32-32-32h-96l-7.2-14.3C307.4 6.8 296.3 0 284.2 0H163.8c-12.1 0-23.2 6.8-28.6 17.7zM416 128H32L53.2 467c1.6 25.3 22.6 45 47.9 45H346.9c25.3 0 46.3-19.7 47.9-45L416 128z"/></svg> Delete</button>';
    document.body.appendChild(contextMenuEl);
  }
}

document.addEventListener('contextmenu', function(e) {
  var item = e.target.closest('[data-message-id]');
  if (!item) {
    if (contextMenuEl) contextMenuEl.style.display = 'none';
    return;
  }
  e.preventDefault();
  ensureContextMenu();
  contextMessageId = item.getAttribute('data-message-id');
  contextMenuEl.style.left = Math.min(e.clientX, window.innerWidth - 190) + 'px';
  contextMenuEl.style.top = Math.min(e.clientY, window.innerHeight - 60) + 'px';
  contextMenuEl.style.display = 'block';
});

document.addEventListener('click', function(e) {
  if (contextMenuEl && !contextMenuEl.contains(e.target)) {
    contextMenuEl.style.display = 'none';
  }
});

function deleteContextMenuItem() {
  if (!contextMessageId) return;
  var id = contextMessageId;
  if (contextMenuEl) contextMenuEl.style.display = 'none';
  showConfirmModal({ title: 'Delete', message: 'Remove this item from chat?', confirmClass: 'bg-[var(--color-error)] text-white', confirmText: 'Delete' }).then(function(confirmed) {
    if (!confirmed) return;
    fetch('/business/messages/' + id, {
      method: 'DELETE',
      headers: { 'X-CSRF-Token': getCookie('csrf_token') }
    })
    .then(function(r) { return r.json(); })
    .then(function(data) {
      if (data.success) {
        showNotification('Deleted', 'success');
        reloadBusinessChatFromServer();
      } else {
        showNotification(data.error || 'Failed to delete', 'error');
      }
    })
    .catch(function(e) { console.error(e); showNotification('Failed to delete', 'error'); });
  });
}

function markMessageRead() {
  if (!contextMessageId) return;
  var id = contextMessageId;
  if (contextMenuEl) contextMenuEl.style.display = 'none';
  if (id >= 10000) {
    showNotification('Only text messages can be marked as read', 'info');
    return;
  }
  fetch('/business/messages/' + id + '/read', {
    method: 'PUT',
    headers: { 'X-CSRF-Token': getCookie('csrf_token') }
  })
  .then(function(r) { return r.json(); })
  .then(function(data) {
    if (data.success) {
      showNotification('Marked as read', 'success');
    } else {
      showNotification(data.error || 'Failed to mark as read', 'error');
    }
  })
  .catch(function(e) { console.error(e); showNotification('Failed to mark as read', 'error'); });
}

window.addEventListener('beforeunload', function() {
  if (wsClient) wsClient.disconnect();
});

document.addEventListener('click', function(e) {
  const progressBtn = e.target.closest('.view-chat-progress-btn');
  if (progressBtn) {
    const clientId = progressBtn.getAttribute('data-client-id');
    showConversationProgress(clientId);
  }

});

function showConversationProgress(clientId) {
  fetch('/business/clients/' + clientId + '/conversation-id')
    .then(response => response.json())
    .then(data => {
      if (data.conversation_id) {
        htmx.ajax('GET', '/conversations/' + data.conversation_id + '/progress', {
          target: '#progress-modal',
          swap: 'innerHTML'
        });
        showProgressModal();
      }
    })
    .catch(console.error);
}

function showProgressModal() {
  if (!document.getElementById('progress-modal')) {
    const modal = document.createElement('div');
    modal.id = 'progress-modal';
    modal.className = 'fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50';
    document.body.appendChild(modal);
  }
}

// ========== Customer Insights ==========

function toggleInsightsDrawer(conversationId) {
  var drawer = document.getElementById('insights-drawer');
  if (!drawer) return;
  
  if (!drawer.classList.contains('open')) {
    positionInsightsDrawer();
    drawer.classList.add('open');
    if (!drawer.hasChildNodes() || drawer.innerHTML.trim() === '') {
      drawer.innerHTML = '<div class="px-3 sm:px-6 py-6 text-center text-[var(--color-text-muted)] text-sm"><i class="fas fa-spinner fa-spin mr-2"></i>Loading insights...</div>';
      htmx.ajax('GET', '/business/conversations/' + conversationId + '/insights-panel', {
        target: '#insights-drawer',
        swap: 'innerHTML'
      });
    }
  } else {
    closeInsightsDrawer();
  }
}

function positionInsightsDrawer() {
  var container = document.getElementById('waChatContainer');
  var drawer = document.getElementById('insights-drawer');
  if (!container || !drawer) return;

  var containerRect = container.getBoundingClientRect();
  var progress = container.querySelector('.wa-progress-bar');
  var input = container.querySelector('.wa-input-wrapper');
  var top = progress ? progress.getBoundingClientRect().bottom - containerRect.top : 0;
  var bottom = input ? containerRect.bottom - input.getBoundingClientRect().top : 0;

  drawer.style.setProperty('--insights-top', Math.max(0, Math.round(top)) + 'px');
  drawer.style.setProperty('--insights-bottom', Math.max(0, Math.round(bottom)) + 'px');
}

function closeInsightsDrawer() {
  var drawer = document.getElementById('insights-drawer');
  if (drawer) drawer.classList.remove('open');
}

window.addEventListener('resize', function() {
  var drawer = document.getElementById('insights-drawer');
  if (drawer && drawer.classList.contains('open')) positionInsightsDrawer();
});

document.addEventListener('click', function(event) {
  var drawer = document.getElementById('insights-drawer');
  if (!drawer || !drawer.classList.contains('open')) return;
  if (event.target.closest('#insights-drawer') || event.target.closest('.insights-toggle')) return;
  closeInsightsDrawer();
});

// ========== Order Lifecycle Functions ==========

function sendOrderToClient(orderId) {
  fetch(`/business/orders/${orderId}/send`, { method: 'POST', headers: { 'X-CSRF-Token': getCookie('csrf_token') } })
    .then(r => r.json())
    .then(data => {
      if (data.success) {
        showNotification('Order sent to client!', 'success');
        reloadBusinessChatFromServer();
      } else {
        showNotification(data.error || 'Failed to send order', 'error');
      }
    })
    .catch(e => { console.error(e); showNotification('Failed to send order', 'error'); });
}

function confirmOrderBusiness(orderId) {
  showConfirmModal({ title: 'Confirm Order', message: 'Confirm this order?' }).then(function(confirmed) {
    if (!confirmed) return;
    fetch(`/business/orders/${orderId}/confirm`, { method: 'POST', headers: { 'X-CSRF-Token': getCookie('csrf_token') } })
    .then(r => r.json())
    .then(data => {
      if (data.success) {
        showNotification('Order confirmed!', 'success');
        reloadBusinessChatFromServer();
      } else {
        showNotification(data.error || 'Failed to confirm order', 'error');
      }
    })
    .catch(e => { console.error(e); showNotification('Failed to confirm order', 'error'); });
  });
}

function rejectOrder(orderId) {
  showConfirmModal({ title: 'Reject Order', message: 'Reject this order?', confirmClass: 'bg-[var(--color-error)] text-white', confirmText: 'Reject' }).then(function(confirmed) {
    if (!confirmed) return;
    fetch(`/business/orders/${orderId}/reject`, { method: 'POST', headers: { 'X-CSRF-Token': getCookie('csrf_token') } })
    .then(r => r.json())
    .then(data => {
      if (data.success) {
        showNotification('Order rejected', 'info');
        reloadBusinessChatFromServer();
      } else {
        showNotification(data.error || 'Failed to reject order', 'error');
      }
    })
    .catch(e => { console.error(e); showNotification('Failed to reject order', 'error'); });
  });
}

function confirmAllOrderPayments(orderId) {
  showConfirmModal({ title: 'Confirm Payments', message: 'Confirm all pending payments for this order?' }).then(function(confirmed) {
    if (!confirmed) return;
    fetch(`/business/orders/${orderId}/payments/confirm-all`, { method: 'POST', headers: { 'X-CSRF-Token': getCookie('csrf_token') } })
    .then(r => r.json())
    .then(data => {
      if (data.success) {
        showNotification(data.message || 'Payments confirmed!', 'success');
        reloadBusinessChatFromServer();
      } else {
        showNotification(data.error || 'Failed to confirm payments', 'error');
      }
    })
    .catch(e => { console.error(e); showNotification('Failed to confirm payments', 'error'); });
  });
}

function confirmAllBookingPayments(bookingId) {
  showConfirmModal({ title: 'Confirm Payments', message: 'Confirm all pending payments for this booking?' }).then(function(confirmed) {
    if (!confirmed) return;
    fetch(`/business/bookings/${bookingId}/payments/confirm-all`, { method: 'POST', headers: { 'X-CSRF-Token': getCookie('csrf_token') } })
    .then(r => r.json())
    .then(data => {
      if (data.success) {
        showNotification(data.message || 'Payments confirmed!', 'success');
        reloadBusinessChatFromServer();
      } else {
        showNotification(data.error || 'Failed to confirm payments', 'error');
      }
    })
    .catch(e => { console.error(e); showNotification('Failed to confirm payments', 'error'); });
  });
}

function fulfillOrder(orderId) {
  showConfirmModal({ title: 'Complete Order', message: 'Mark this order as completed?' }).then(function(confirmed) {
    if (!confirmed) return;
    fetch(`/business/orders/${orderId}/fulfill`, { method: 'POST', headers: { 'X-CSRF-Token': getCookie('csrf_token') } })
    .then(r => r.json())
    .then(data => {
      if (data.success) {
        showNotification('Order completed!', 'success');
        reloadBusinessChatFromServer();
      } else {
        showNotification(data.error || 'Failed to complete order', 'error');
      }
    })
    .catch(e => { console.error(e); showNotification('Failed to complete order', 'error'); });
  });
}

function cancelDraftOrder(orderId) {
  showConfirmModal({ title: 'Discard Draft', message: 'Discard this draft order?', confirmClass: 'bg-[var(--color-error)] text-white', confirmText: 'Discard' }).then(function(confirmed) {
    if (!confirmed) return;
    fetch(`/business/orders/${orderId}/reject`, { method: 'POST', headers: { 'X-CSRF-Token': getCookie('csrf_token') } })
    .then(r => r.json())
    .then(data => {
      if (data.success) {
        showNotification('Draft discarded', 'info');
        reloadBusinessChatFromServer();
      } else {
        showNotification(data.error || 'Failed to discard draft', 'error');
      }
    })
    .catch(e => { console.error(e); showNotification('Failed to discard draft', 'error'); });
  });
}

function updateBookingStatusFromCard(bookingId, newStatus) {
  const action = newStatus === 'client_confirmed' ? 'confirm' : newStatus === 'completed' ? 'complete' : 'cancel';
  showConfirmModal({ title: action.charAt(0).toUpperCase() + action.slice(1) + ' Booking', message: 'Are you sure you want to ' + action + ' this booking?', confirmText: action.charAt(0).toUpperCase() + action.slice(1), confirmClass: newStatus === 'cancelled' ? 'bg-[var(--color-error)] text-white' : 'bg-[var(--color-primary)] text-white' }).then(function(confirmed) {
    if (!confirmed) return;

    fetch(`/business/bookings/${bookingId}/status`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': getCookie('csrf_token') },
    body: JSON.stringify({ status: newStatus })
  })
    .then(r => r.json())
    .then(data => {
      if (data.success) {
        showNotification(`Booking ${action}ed successfully!`, 'success');
        reloadBusinessChatFromServer();
      } else {
        showNotification(data.error || `Failed to ${action} booking`, 'error');
      }
    })
    .catch(e => { console.error(e); showNotification(`Failed to ${action} booking`, 'error'); });
  });
}

// ========== Quick Replies & Input Handling ==========

function onMessageInput(input) {
  var val = input.value;

  // Show quick replies when typing /
  var qr = document.getElementById('quickReplies');
  if (qr) {
    if (val === '/') {
      qr.classList.remove('hidden');
    } else if (qr && !qr.classList.contains('hidden') && val.charAt(0) !== '/') {
      qr.classList.add('hidden');
    }
  }

  // Typing indicator via WebSocket
  if (wsClient && wsClient.isConnected) {
    if (typingTimeout) clearTimeout(typingTimeout);
    if (val.length > 0) {
      wsClient.sendTypingStart(conversationId, businessId, 'business', clientId, businessId);
    } else {
      wsClient.sendTypingStop(conversationId, businessId, 'business', clientId, businessId);
    }
    typingTimeout = setTimeout(function() {
      wsClient.sendTypingStop(conversationId, businessId, 'business', clientId, businessId);
    }, 3000);
  }
}

function onMessageKeydown(event) {
  var qr = document.getElementById('quickReplies');
  if (event.key === 'Escape' && qr && !qr.classList.contains('hidden')) {
    qr.classList.add('hidden');
    var input = document.getElementById('messageInput');
    if (input) input.value = input.value.replace(/\/$/, '');
  }

  // Send typing stop on Enter (message sent)
  if (event.key === 'Enter' && !event.shiftKey) {
    if (wsClient && wsClient.isConnected) {
      wsClient.sendTypingStop(conversationId, businessId, 'business', clientId, businessId);
    }
    if (typingTimeout) {
      clearTimeout(typingTimeout);
      typingTimeout = null;
    }
  }
}
