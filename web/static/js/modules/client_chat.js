function hideClientOrderModal() {
  document.getElementById('clientOrderModal').classList.add('hidden');
  document.getElementById('clientOrderForm').reset();
}

function submitOrderForm() {
  const productSelect = document.getElementById('clientOrderProduct');
  const quantityInput = document.getElementById('clientOrderQuantity');
  const addressInput = document.getElementById('clientOrderAddress');
  const notesInput = document.getElementById('clientOrderNotes');

  if (!productSelect.value) return showNotification('Please select a product', 'error');
  if (!quantityInput.value || quantityInput.value < 1) return showNotification('Please enter a valid quantity', 'error');

  const data = {
    product_id: parseInt(productSelect.value),
    quantity: parseInt(quantityInput.value),
    delivery_address: addressInput.value,
    notes: notesInput.value,
    business_id: parseInt(businessId)
  };

  fetch('/client/orders', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': getCookie('csrf_token') },
    body: JSON.stringify(data)
  })
    .then(r => r.json())
    .then(data => {
      if (data.success) {
        hideClientOrderModal();
        showNotification('Order request sent successfully!', 'success');
      } else {
        showNotification(data.error || 'Failed to send order request', 'error');
      }
    })
    .catch(e => { console.error(e); showNotification('Failed to send order request', 'error'); });
}

if (window.typingTimeout) clearTimeout(window.typingTimeout);
var typingTimeout = null;

var unreadBelow = 0;
window.clearUnreadBelow = function() {
  if (unreadBelow > 0 && typeof markAsRead === 'function') markAsRead();
  unreadBelow = 0;
  var badge = document.getElementById('scrollBottomBadge');
  if (badge) badge.classList.remove('visible');
};

function updateScrollBottomBadge() {
  var badge = document.getElementById('scrollBottomBadge');
  if (!badge) return;
  if (unreadBelow > 0) {
    badge.textContent = unreadBelow > 99 ? '99+' : unreadBelow;
    badge.classList.add('visible');
  } else {
    badge.classList.remove('visible');
  }
}

function updateSidebarCard(frame) {
  if (frame.sender_type !== 'business') return;
  var msg = frame.new_message;
  if (!msg) return;
  if (msg.msg_type === 'order' || msg.msg_type === 'booking') return;

  var bid = frame.sender_id;
  var item = document.querySelector('.business-item[data-business-id="' + bid + '"]');
  if (!item) return;

  // Update preview text
  var preview = item.querySelector('.wa-chat-preview');
  if (preview) {
    if (msg.media_url) {
      preview.textContent = 'Media';
    } else if (msg.content) {
      preview.textContent = msg.content.length > 60 ? msg.content.substring(0, 57) + '...' : msg.content;
    }
  }

  // Update timestamp
  var timeEl = item.querySelector('.wa-chat-time.time-ago');
  if (timeEl && msg.created_at) {
    var iso = new Date(Number(msg.created_at)).toISOString();
    timeEl.setAttribute('data-time', iso);
  }

  // Increment unread badge
  var badge = item.querySelector('.wa-unread-badge');
  if (badge) {
    var count = parseInt(badge.textContent) + 1;
    badge.textContent = count > 99 ? '99+' : count;
  } else {
    var bottom = item.querySelector('.wa-chat-bottom');
    if (bottom) {
      bottom.insertAdjacentHTML('beforeend', '<span class="wa-unread-badge">1</span>');
    }
  }

  // Reorder card to top
  var list = item.parentElement;
  if (list && list.firstChild !== item) {
    list.insertBefore(item, list.firstChild);
  }
}

scrollToBottom();
markAsRead();
startWsClient();

function scrollToBottom() {
  var container = document.getElementById('messages-container');
  if (container) requestAnimationFrame(function () {
    container.scrollTop = container.scrollHeight;
  });
}

window.scrollToBottomBtn = function() {
  clearUnreadBelow();
  scrollToBottom();
};

function markAsRead() {
  fetch(`/client/businesses/${businessId}/read`, { method: 'PUT', headers: { 'X-CSRF-Token': getCookie('csrf_token') } })
    .then(function () {
      var badge = document.querySelector('.business-item[data-business-id="' + businessId + '"] .wa-unread-badge');
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
  if (!receipt || receipt.reader_type === 'client') return;
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

function reloadClientChatFromServer() {
  if (!businessId) return;
  fetch('/client/businesses/' + businessId + '/messages')
    .then(function(r) { return r.text(); })
    .then(function(html) {
      var parser = new DOMParser();
      var doc = parser.parseFromString(html, 'text/html');
      var next = doc.getElementById('chat-content') || doc.getElementById('messages-container');
      var current = document.getElementById(next && next.id === 'chat-content' ? 'chat-content' : 'messages-container');
      if (next && current) {
        current.innerHTML = next.innerHTML;
      }
    })
    .catch(console.error);
}

function starsHtml(rating) {
  var html = '';
  var r = rating || 5;
  for (var i = 1; i <= 5; i++) {
    html += i <= r ? '<i class="fas fa-star"></i>' : '<i class="far fa-star"></i>';
  }
  return html;
}

function addReviewBadgeToCard(card, rating) {
  if (!card || card.querySelector('[data-review-badge]')) return;
  var badge = document.createElement('div');
  badge.setAttribute('data-review-badge', '1');
  badge.className = 'w-full mt-2 py-1.5 px-3 rounded-lg bg-[var(--color-warning-light)] text-[var(--color-warning)] text-xs font-medium text-center flex items-center justify-center gap-0.5';
  badge.innerHTML = starsHtml(rating) + '<span class="ml-1.5 text-[var(--color-text-secondary)]">Reviewed</span>';
  var timestamp = card.querySelector('.mt-2.text-right');
  if (timestamp) card.insertBefore(badge, timestamp);
  else card.appendChild(badge);
}

function applyClientOrderCardUpdate(upd) {
  if (!upd || !upd.order_id) return false;
  var card = document.querySelector('[data-order-id="' + upd.order_id + '"]');
  if (!card) {
    if (upd.card_html) {
      var container = document.getElementById('messages-container');
      if (container) {
        var isNearBottom = container.scrollHeight - container.scrollTop - container.clientHeight < 150;
        container.insertAdjacentHTML('beforeend', upd.card_html);
        if (isNearBottom) container.scrollTop = container.scrollHeight;
        return true;
      }
    }
    return false;
  }

  if (upd.card_html) {
    var container = document.getElementById('messages-container');
    var scrollTop = container ? container.scrollTop : 0;
    card.outerHTML = upd.card_html;
    if (container && container.scrollTop !== scrollTop) {
      requestAnimationFrame(function() { container.scrollTop = scrollTop; });
    }
    return true;
  }

  if (upd.status && card.getAttribute('data-order-status') && upd.status !== card.getAttribute('data-order-status')) {
    return false;
  }
  if (upd.has_review) {
    addReviewBadgeToCard(card, upd.review_rating || 5);
    return true;
  }
  return false;
}

function applyClientBookingCardUpdate(upd) {
  if (!upd || !upd.booking_id) return false;
  var card = document.querySelector('[data-booking-id="' + upd.booking_id + '"]');
  if (!card) {
    if (upd.card_html) {
      var container = document.getElementById('messages-container');
      if (container) {
        var isNearBottom = container.scrollHeight - container.scrollTop - container.clientHeight < 150;
        container.insertAdjacentHTML('beforeend', upd.card_html);
        if (isNearBottom) container.scrollTop = container.scrollHeight;
        return true;
      }
    }
    return false;
  }

  if (upd.card_html) {
    var container = document.getElementById('messages-container');
    var scrollTop = container ? container.scrollTop : 0;
    card.outerHTML = upd.card_html;
    if (container && container.scrollTop !== scrollTop) {
      requestAnimationFrame(function() { container.scrollTop = scrollTop; });
    }
    return true;
  }

  if (upd.status && card.getAttribute('data-booking-status') && upd.status !== card.getAttribute('data-booking-status')) {
    return false;
  }
  if (upd.has_review) {
    addReviewBadgeToCard(card, upd.review_rating || 5);
    return true;
  }
  return false;
}

function startWsClient() {
  if (!window.wsClient || !window.wsClient.isConnected) {
    window.wsClient = new WsClient();
    var token = getCookie('client_token');
    if (!token) return;
    window.wsClient.connect('/ws/client?token=' + encodeURIComponent(token));
  }
  registerClientChatHandlers();
}

function registerClientChatHandlers() {
  if (window._clientChatHandlersRegistered) return;
  window._clientChatHandlersRegistered = true;

  window.wsClient.on(1, function(frame) {
    var msg = frame.new_message;
    if (!msg) return;
    if (msg.msg_type === 'order' || msg.msg_type === 'booking') return;

    // Send delivery ack for every received message (WhatsApp-style)
    if (window.wsClient && frame.conversation_id) {
      window.wsClient.sendDeliveredAck(frame.conversation_id, frame.sender_id || '');
    }

    // Update sidebar card (preview, time, reorder, badge) for all received messages
    if (frame.conversation_id) {
      updateSidebarCard(frame);
    }

    // Message for a different conversation — stop here (don't render in current chat)
    if (frame.conversation_id && frame.conversation_id !== String(conversationId)) {
      return;
    }

    var container = document.getElementById('messages-container');
    if (!container) return;
    if (frame.sender_type === 'client') return;
    var html = '';
    if (msg.media_url) {
      html = renderMediaMessage(msg);
    } else {
      html = '<div class="msg in message-item" data-message-id="' + msg.id + '"><div class="msg-bbl"><svg class="msg-tail" viewBox="0 0 10 15" height="15" width="10" preserveAspectRatio="xMidYMid meet"><path fill="var(--color-bg)" d="M1,3L10,14V1H3C1.5,1,0.5,2,1,3z"></path><path fill="currentColor" d="M1,2L10,13V0H3C1.5,0,0.5,1,1,2z"></path></svg><span class="msg-txt">' + escapeHtml(msg.content || '') + '</span><span class="msg-meta"><span class="msg-time">' + formatTime(msg.created_at) + '</span></span></div></div>';
    }
    container.insertAdjacentHTML('beforeend', html);

    var isNearBottom = container.scrollHeight - container.scrollTop - container.clientHeight < 150;
    if (isNearBottom) {
      container.scrollTop = container.scrollHeight;
    } else {
      unreadBelow += 1;
      updateScrollBottomBadge();
    }

    markVisibleConversationRead();
  });

  window.wsClient.on(3, function(frame) {
    showTypingIndicator(frame.typing);
  });
  window.wsClient.on(4, function(frame) {
    hideTypingIndicator(frame.typing);
  });

  window.wsClient.on(6, function(frame) {
    var upd = frame.order_update;
    if (!upd) return;
    if (!applyClientOrderCardUpdate(upd)) {}
  });

  window.wsClient.on(7, function(frame) {
    var upd = frame.booking_update;
    if (!upd) return;
    if (!applyClientBookingCardUpdate(upd)) {}
  });

  window.wsClient.on(8, function(frame) {
    if (!frame.unread_count) return;
    var uc = frame.unread_count;
    if (!uc.conversation_id) return;
    var item = document.querySelector('.business-item[data-conversation-id="' + uc.conversation_id + '"]');
    if (!item) return;
    var badge = item.querySelector('.wa-unread-badge');
    if (uc.count > 0) {
      if (badge) {
        badge.textContent = uc.count > 99 ? '99+' : uc.count;
      } else {
        var bottom = item.querySelector('.wa-chat-bottom');
        if (bottom) {
          bottom.insertAdjacentHTML('beforeend', '<span class="wa-unread-badge">' + (uc.count > 99 ? '99+' : uc.count) + '</span>');
        }
      }
    } else {
      if (badge) badge.remove();
    }
  });

  window.wsClient.on(2, function(frame) {
    applyReadReceipt(frame.read_receipt);
  });

  window.wsClient.on(5, function(frame) {
    // Presence updates not used on client chat page
  });

  window.wsClient.on(12, function(frame) {
    if (!frame.delivered_receipt) return;
    var dr = frame.delivered_receipt;
    if (dr.conversation_id && dr.conversation_id !== String(conversationId)) return;
    document.querySelectorAll('#messages-container .message-item.out').forEach(function(item) {
      var tick = item.querySelector('.msg-tick');
      if (!tick) return;
      if (tick.getAttribute('data-read-state') === 'read') return;
      if (tick.getAttribute('data-read-state') === 'delivered') return;
      tick.setAttribute('data-read-state', 'delivered');
      tick.innerHTML = '<svg viewBox="0 0 16 12" width="14" height="11" fill="none" stroke="#8696a0" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M2 6L5 9L11 3"/><path d="M6 6L9 9L15 3"/></svg>';
      tick.style.width = '14px';
    });
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
  if (typing.user_type === 'client') return;
  var el = document.getElementById('typingIndicator');
  if (!el) {
    el = document.createElement('div');
    el.id = 'typingIndicator';
    el.className = 'msg in typing-indicator';
    el.innerHTML = '<div class="msg-bbl typing"><span class="typing-label">typing</span><span class="typing-dot"></span><span class="typing-dot"></span><span class="typing-dot"></span></div>';
    document.getElementById('messages-container').appendChild(el);
  }
}

function hideTypingIndicator(typing) {
  if (!typing || typing.conversation_id !== String(conversationId)) return;
  if (typing.user_type === 'client') return;
  var el = document.getElementById('typingIndicator');
  if (el) el.remove();
}

function addOrderMessageToChat(order) {
  const container = document.getElementById('messages-container');
  if (!container) return;
  const status = order.status || 'pending';
  const bgClass = status === 'pending' ? 'bg-[var(--color-warning-light)] border-[var(--color-warning)]' :
    status === 'client_confirmed' ? 'bg-[var(--color-info-light)] border-[var(--color-info)]' :
    status === 'confirmed' ? 'bg-[var(--color-info-light)] border-[var(--color-info)]' :
    status === 'fulfilled' || status === 'completed' ? 'bg-[var(--color-success-light)] border-[var(--color-success)]' :
    status === 'cancelled' ? 'bg-[var(--color-error-light)] border-[var(--color-error)]' : 'bg-[var(--color-info-light)] border-[var(--color-info)]';
  const iconColor = status === 'pending' ? 'text-[var(--color-warning)]' :
    status === 'client_confirmed' ? 'text-[var(--color-info)]' :
    status === 'confirmed' ? 'text-[var(--color-info)]' :
    status === 'fulfilled' || status === 'completed' ? 'text-[var(--color-success)]' :
    status === 'cancelled' ? 'text-[var(--color-error)]' : 'text-[var(--color-info)]';
  const statusLabel = status === 'pending' ? 'Pending' :
    status === 'client_confirmed' ? 'Confirmed' :
    status === 'confirmed' ? 'Confirmed' :
    status === 'fulfilled' || status === 'completed' ? 'Completed' :
    status === 'cancelled' ? 'Cancelled' : 'Pending';
  const statusBadgeBg = status === 'pending' ? 'bg-[var(--color-warning-light)] text-[var(--color-warning)]' :
    status === 'client_confirmed' ? 'bg-[var(--color-info-light)] text-[var(--color-info)]' :
    status === 'confirmed' ? 'bg-[var(--color-info-light)] text-[var(--color-info)]' :
    status === 'fulfilled' || status === 'completed' ? 'bg-[var(--color-success-light)] text-[var(--color-success)]' :
    status === 'cancelled' ? 'bg-[var(--color-error-light)] text-[var(--color-error)]' : 'bg-[var(--color-warning-light)] text-[var(--color-warning)]';
  const div = document.createElement('div');
  div.className = 'flex justify-end';
  div.innerHTML = `<div class="max-w-xs lg:max-w-md w-full">
    <div class="${bgClass} border rounded-lg px-4 py-3" data-message-id="${order.id}" data-order-id="${order.id}">
      <div class="flex items-center justify-between mb-2">
        <div class="flex items-center space-x-2">
          <i class="fas fa-shopping-cart ${iconColor}"></i>
          <span class="font-semibold ${iconColor} text-sm">[${order.id}]</span>
          <span class="text-[var(--color-text)] text-sm">${order.product_name || 'Product'}</span>
        </div>
        <button onclick="openClientEditOrderPicker(${order.id})" class="${iconColor} hover:opacity-80 text-xs" title="Edit Order">
          <i class="fas fa-edit"></i>
        </button>
      </div>
      <div class="order-details text-sm text-[var(--color-text)]">
        <p class="text-sm">Order #${order.order_number} - ${order.quantity || 1}x - $${parseFloat(order.total_amount).toFixed(2)}</p>
        <p class="hidden order-notes-data">${order.notes || ''}</p>
      </div>
      <div class="flex items-center justify-between mt-2">
        <p class="text-xs text-[var(--color-text-muted)]">${new Date().toLocaleTimeString('en-US', {hour:'numeric', minute:'2-digit'})}</p>
        <span class="text-xs ${statusBadgeBg} px-2 py-1 rounded">${statusLabel}</span>
      </div>
    </div>
  </div>`;
  container.appendChild(div);
  container.scrollTop = container.scrollHeight;
}

function clientConfirmOrder(orderId) {
  showConfirmModal({ title: 'Confirm Order', message: 'Confirm this order?' }).then(function(confirmed) {
    if (!confirmed) return;
    fetch(`/client/orders/${orderId}/confirm`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': getCookie('csrf_token') },
    body: JSON.stringify({ items: [] })
  })
    .then(r => r.json())
    .then(data => {
      if (data.success) {
        showNotification(data.message || 'Order confirmed!', 'success');
      } else {
        showNotification(data.error || 'Failed to confirm order', 'error');
      }
    })
    .catch(e => { console.error(e); showNotification('Failed to confirm order', 'error'); });
  });
}

function clientOrderItemIncrement(orderId, productId, btn) {
  const qtySpan = btn.parentElement.querySelector('.qty-value');
  const current = parseInt(qtySpan.textContent);
  qtySpan.textContent = current + 1;
  updateClientOrderTotal(orderId);
}

function clientOrderItemDecrement(orderId, productId, btn) {
  const qtySpan = btn.parentElement.querySelector('.qty-value');
  const current = parseInt(qtySpan.textContent);
  if (current > 1) {
    qtySpan.textContent = current - 1;
  }
  updateClientOrderTotal(orderId);
}

function updateClientOrderTotal(orderId) {
  const card = document.querySelector(`[data-order-id="${orderId}"]`);
  if (!card) return;
  let total = 0;
  card.querySelectorAll('[data-item-product-id]').forEach(item => {
    const qty = parseInt(item.querySelector('.qty-value').textContent);
    const priceEl = item.closest('.flex.items-center.justify-between').querySelector('.text-sm.font-bold');
    const priceText = priceEl ? priceEl.textContent.replace(/[^0-9.]/g, '') : '0';
    total += qty * parseFloat(priceText);
  });
  const totalEl = card.querySelector('.text-lg.font-bold');
  if (totalEl) totalEl.textContent = (typeof currencySymbol !== 'undefined' ? currencySymbol : '$') + total.toFixed(2);
}

function cancelOrder(orderId) {
  showConfirmModal({ title: 'Cancel Order', message: 'Are you sure you want to cancel this order?', confirmClass: 'bg-[var(--color-error)] text-white', confirmText: 'Cancel Order' }).then(function(confirmed) {
    if (!confirmed) return;
    fetch(`/client/orders/${orderId}/cancel`, {
    method: 'POST',
    headers: { 'Authorization': 'Bearer ' + getCookie('client_token'), 'X-CSRF-Token': getCookie('csrf_token') }
  })
    .then(r => r.json())
    .then(data => {
      if (data.success) {
        showNotification('Order cancelled successfully', 'success');
      } else {
        showNotification(data.error || 'Failed to cancel order', 'error');
      }
    })
    .catch(e => { console.error(e); showNotification('Failed to cancel order', 'error'); });
  });
}

function cancelBooking(bookingId) {
  showConfirmModal({ title: 'Cancel Booking', message: 'Are you sure you want to cancel this booking?', confirmClass: 'bg-[var(--color-error)] text-white', confirmText: 'Cancel Booking' }).then(function(confirmed) {
    if (!confirmed) return;
  fetch(`/client/bookings/${bookingId}/cancel`, {
    method: 'POST',
    headers: { 'X-CSRF-Token': getCookie('csrf_token') }
  })
    .then(r => r.json())
    .then(data => {
      if (data.success) {
        showNotification('Booking cancelled successfully', 'success');
      } else {
        showNotification(data.error || 'Failed to cancel booking', 'error');
      }
    })
    .catch(e => { console.error(e); showNotification('Failed to cancel booking', 'error'); });
  });
}

function clientConfirmBooking(bookingId) {
  showConfirmModal({ title: 'Approve Booking', message: 'Are you sure you want to approve this booking?', confirmText: 'Approve', confirmClass: 'bg-[var(--color-success)] text-white' }).then(function(confirmed) {
    if (!confirmed) return;
  fetch(`/client/bookings/${bookingId}/confirm`, {
    method: 'POST',
    headers: { 'X-CSRF-Token': getCookie('csrf_token') }
  })
    .then(r => r.json())
    .then(data => {
      if (data.success) {
        showNotification('Booking confirmed!', 'success');
      } else {
        showNotification(data.error || 'Failed to confirm booking', 'error');
      }
    })
    .catch(e => { console.error(e); showNotification('Failed to confirm booking', 'error'); });
  });
}

// ========== Typing Indicator ==========

var messageInput = document.getElementById('messageInput');
if (messageInput) {
  messageInput.addEventListener('input', function() {
    if (window.wsClient && window.wsClient.isConnected) {
      if (typingTimeout) clearTimeout(typingTimeout);
      if (this.value.length > 0) {
        window.wsClient.sendTypingStart(conversationId, clientId, 'client', clientId, businessId);
      } else {
        window.wsClient.sendTypingStop(conversationId, clientId, 'client', clientId, businessId);
      }
      typingTimeout = setTimeout(function() {
        window.wsClient.sendTypingStop(conversationId, clientId, 'client', clientId, businessId);
      }, 3000);
    }
  });

  messageInput.addEventListener('keydown', function(event) {
    if (event.key === 'Enter' && !event.shiftKey) {
      if (window.wsClient && window.wsClient.isConnected) {
        window.wsClient.sendTypingStop(conversationId, clientId, 'client', clientId, businessId);
      }
      if (typingTimeout) {
        clearTimeout(typingTimeout);
        typingTimeout = null;
      }
    }
  });
}

function toggleMediaTray() {
  var tray = document.getElementById('media-tray');
  var icon = document.getElementById('media-icon');
  if (tray) {
    tray.classList.toggle('hidden');
    if (icon) {
      icon.classList.toggle('fa-paperclip');
      icon.classList.toggle('fa-times');
    }
  }
}

function triggerMediaUpload(type) {
  var input = document.getElementById('media-input-' + type);
  if (input) input.click();
  var tray = document.getElementById('media-tray');
  if (tray && !tray.classList.contains('hidden')) {
    tray.classList.add('hidden');
    var icon = document.getElementById('media-icon');
    if (icon) icon.classList.replace('fa-times', 'fa-paperclip');
  }
}

function handleMediaSelected(input) {
  if (input.files && input.files.length > 0) {
    var form = document.getElementById('message-form');
    var textInput = form ? form.querySelector('input[name="content"]') : null;
    if (textInput) textInput.required = false;
    if (form && form.requestSubmit) {
      form.requestSubmit();
    } else if (form) {
      form.submit();
    }
    if (textInput) textInput.required = true;
  }
}

document.addEventListener('click', function(e) {
  var container = document.getElementById('media-tray-container');
  var tray = document.getElementById('media-tray');
  if (container && tray && !tray.classList.contains('hidden') && !container.contains(e.target)) {
    tray.classList.add('hidden');
    var icon = document.getElementById('media-icon');
    if (icon) {
      icon.classList.replace('fa-times', 'fa-paperclip');
    }
  }
});

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

function markMessageRead() {
  if (!contextMessageId) return;
  if (contextMenuEl) contextMenuEl.style.display = 'none';
  fetch('/client/businesses/' + businessId + '/read', {
    method: 'PUT',
    headers: { 'X-CSRF-Token': getCookie('csrf_token') }
  })
  .then(function(r) { return r.json(); })
  .then(function(data) {
    if (data.status === 'ok') {
      showNotification('Conversation marked as read', 'success');
      var badge = document.querySelector('.business-item[data-business-id="' + businessId + '"] .unread-badge');
      if (badge) badge.remove();
    } else {
      showNotification('Failed to mark as read', 'error');
    }
  })
  .catch(function(e) { console.error(e); showNotification('Failed to mark as read', 'error'); });
}

function deleteContextMenuItem() {
  if (!contextMessageId) return;
  var id = contextMessageId;
  if (contextMenuEl) contextMenuEl.style.display = 'none';
  showConfirmModal({ title: 'Delete', message: 'Remove this item from chat?', confirmClass: 'bg-[var(--color-error)] text-white', confirmText: 'Delete' }).then(function(confirmed) {
    if (!confirmed) return;
    fetch('/client/messages/' + id, {
      method: 'DELETE',
      headers: { 'X-CSRF-Token': getCookie('csrf_token') }
    })
    .then(function(r) { return r.json(); })
    .then(function(data) {
      if (data.success) {
        showNotification('Deleted', 'success');
        var el = document.querySelector('[data-message-id="' + id + '"]');
        if (el) el.remove();
      } else {
        showNotification(data.error || 'Failed to delete', 'error');
      }
    })
    .catch(function(e) { console.error(e); showNotification('Failed to delete', 'error'); });
  });
}

function onMessageInput(input) {
  var val = input.value;
  if (window.wsClient && window.wsClient.isConnected) {
    if (typingTimeout) clearTimeout(typingTimeout);
    if (val.length > 0) {
      window.wsClient.sendTypingStart(conversationId, clientId, 'client', clientId, businessId);
    } else {
      window.wsClient.sendTypingStop(conversationId, clientId, 'client', clientId, businessId);
    }
    typingTimeout = setTimeout(function() {
      window.wsClient.sendTypingStop(conversationId, clientId, 'client', clientId, businessId);
    }, 3000);
  }
}

function onMessageKeydown(event) {
  if (event.key === 'Enter' && !event.shiftKey) {
    if (window.wsClient && window.wsClient.isConnected) {
      window.wsClient.sendTypingStop(conversationId, clientId, 'client', clientId, businessId);
    }
    if (typingTimeout) {
      clearTimeout(typingTimeout);
      typingTimeout = null;
    }
  }
}

window.addEventListener('beforeunload', function() {
  if (window.wsClient) window.wsClient.disconnect();
});
