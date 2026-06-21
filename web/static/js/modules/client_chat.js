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

// Shared functions from chat_common.js

var _clientSortTimer = null;
function deferredClientSort() {
  if (_clientSortTimer) clearTimeout(_clientSortTimer);
  _clientSortTimer = setTimeout(sortBusinessList, 200);
}

function sortBusinessList() {
  var pins = JSON.parse(localStorage.getItem('pinned_businesses') || '[]');
  var list = document.getElementById('business-list');
  if (!list) return;
  var sentinel = document.getElementById('client-sidebar-sentinel');
  if (sentinel) sentinel.remove();
  list.querySelectorAll('[data-sidebar-hidden]').forEach(function(el) {
    el.removeAttribute('data-sidebar-hidden');
  });
  var items = Array.from(list.querySelectorAll('.business-item'));
  if (items.length < 2) return;
  items.sort(function(a, b) {
    var aP = pins.indexOf(parseInt(a.getAttribute('data-business-id'))) > -1;
    var bP = pins.indexOf(parseInt(b.getAttribute('data-business-id'))) > -1;
    if (aP && !bP) return -1; if (!aP && bP) return 1;
    var aO = a.getAttribute('data-online') === 'true';
    var bO = b.getAttribute('data-online') === 'true';
    if (aO && !bO) return -1; if (!aO && bO) return 1;
    var aU = parseInt(a.getAttribute('data-unread') || '0');
    var bU = parseInt(b.getAttribute('data-unread') || '0');
    if (aU !== bU) return bU - aU;
    var aT = a.getAttribute('data-last-message-at') || '';
    var bT = b.getAttribute('data-last-message-at') || '';
    return bT.localeCompare(aT);
  });
  var parent = list.parentNode;
  var sibling = list.nextSibling;
  parent.removeChild(list);
  items.forEach(function(el) { list.appendChild(el); });
  parent.insertBefore(list, sibling);
  items.forEach(function(el) {
    var star = el.querySelector('.pin-btn svg');
    var btn = el.querySelector('.pin-btn');
    if (star && btn) {
      var id = parseInt(el.getAttribute('data-business-id'));
      var isPinned = pins.indexOf(id) > -1;
      star.outerHTML = isPinned ? heroicon("star", "text-sm", "text-[var(--color-warning)]") : heroicon("star", "text-sm", "text-[var(--color-text-muted)]");
      if (isPinned) {
        btn.classList.add('bg-[var(--color-warning-light)]');
      } else {
        btn.classList.remove('bg-[var(--color-warning-light)]');
      }
    }
  });
  initClientSidebarVirtualScroll();
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
    item.setAttribute('data-last-message-at', iso);
  }

  // Increment unread badge
  var badge = item.querySelector('.wa-unread-badge');
  if (badge) {
    var count = parseInt(badge.textContent) + 1;
    badge.textContent = count > 99 ? '99+' : count;
    item.setAttribute('data-unread', count);
  } else {
    var bottom = item.querySelector('.wa-chat-bottom');
    if (bottom) {
      bottom.insertAdjacentHTML('beforeend', '<span class="wa-unread-badge">1</span>');
    }
    item.setAttribute('data-unread', 1);
  }

  // Reorder card to top
  var list = item.parentElement;
  if (list && list.firstChild !== item) {
    list.insertBefore(item, list.firstChild);
  }

  deferredClientSort();
}

function markAsRead() {
  var container = document.getElementById('messages-container');
  if (!container) return;
  var isNearBottom = container.scrollHeight - container.scrollTop - container.clientHeight < 150;
  if (!isNearBottom) return;

  fetch(`/client/businesses/${businessId}/read`, { method: 'PUT', headers: { 'X-CSRF-Token': getCookie('csrf_token') } })
    .then(function() {
      var badge = document.querySelector('.business-item[data-business-id="' + businessId + '"] .wa-unread-badge');
      if (badge) badge.remove();
    })
    .catch(console.error);
}

scrollToBottom();
startWsClient();

window.scrollToBottomBtn = function() {
  scrollToBottom();
  if (window.clearUnreadBelow) window.clearUnreadBelow();
  if (typeof markAsRead === 'function') markAsRead();
};

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
        initOlderObserver();
        initScrollToBottom();
      }
    })
    .catch(console.error);
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
  if (!window.wsClient) {
    var token = getCookie('client_token');
    if (!token) return;
    window.wsClient = new WsClient();
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
      markVisibleConversationRead();
    } else {
      unreadBelow += 1;
      updateScrollBottomBadge();
    }
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
    item.setAttribute('data-unread', String(uc.count));
    deferredClientSort();
  });

  window.wsClient.on(2, function(frame) {
    applyReadReceipt(frame.read_receipt);
  });

  window.wsClient.on(5, function(frame) {
    var p = frame.presence;
    if (!p) return;
    var bid = frame.sender_id;
    if (!bid) return;
    var el = document.querySelector('.business-item[data-business-id="' + bid + '"]');
    if (el) {
      el.setAttribute('data-online', p.is_online ? 'true' : 'false');
      var dot = el.querySelector('.wa-online-dot');
      if (dot) {
        dot.classList.remove('online', 'offline');
        dot.classList.add(p.is_online ? 'online' : 'offline');
      }
      deferredClientSort();
    }
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

  // CONVERSATION_UPDATE (type 13) — new conversation created/removed, add/remove card on sidebar
  window.wsClient.on(13, function(frame) {
    var update = frame.conversation_update;
    if (!update) return;

    // Handle removal — card deleted
    if (update.removed) {
      var bid = update.client_id || frame.sender_id;
      if (!bid) return;
      var card = document.querySelector('.business-item[data-business-id="' + bid + '"]');
      if (card) card.remove();
      return;
    }

    // Handle insertion / update
    var html = update.client_card_html;
    if (!html) return;

    var parser = new DOMParser();
    var doc = parser.parseFromString(html, 'text/html');
    var card = doc.body.firstElementChild;
    if (!card) return;

    var bid = card.getAttribute('data-business-id');
    if (!bid) return;

    var existing = document.querySelector('.business-item[data-business-id="' + bid + '"]');
    var list = document.getElementById('business-list');
    if (!list) return;

    if (existing) {
      existing.outerHTML = html;
    } else {
      var emptyState = list.querySelector('.wa-empty-chat-list');
      if (emptyState) emptyState.remove();
      list.insertAdjacentHTML('afterbegin', html);
    }
    deferredClientSort();
  });
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
          ${heroicon("shopping-cart", iconColor)}
          <span class="font-semibold ${iconColor} text-sm">[${order.id}]</span>
          <span class="text-[var(--color-text)] text-sm">${order.product_name || 'Product'}</span>
        </div>
        <button onclick="openClientEditOrderPicker(${order.id})" class="${iconColor} hover:opacity-80 text-xs" title="Edit Order">
          ${heroicon("pencil")}
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

// ========== Sidebar Virtual Scrolling ==========

var CLIENT_SIDEBAR_BATCH = 100;
var clientSidebarObserver = null;

function initClientSidebarVirtualScroll() {
  var list = document.getElementById('business-list');
  if (!list) return;
  if (clientSidebarObserver) { clientSidebarObserver.disconnect(); clientSidebarObserver = null; }

  var old = document.getElementById('client-sidebar-sentinel');
  if (old) old.remove();
  list.querySelectorAll('.business-item').forEach(function(el) { el.removeAttribute('data-sidebar-hidden'); });

  var items = list.querySelectorAll('.business-item');
  if (items.length <= CLIENT_SIDEBAR_BATCH) return;

  for (var i = CLIENT_SIDEBAR_BATCH; i < items.length; i++) {
    items[i].setAttribute('data-sidebar-hidden', 'true');
  }

  var sentinel = document.createElement('div');
  sentinel.id = 'client-sidebar-sentinel';
  sentinel.style.height = '1px';
  items[CLIENT_SIDEBAR_BATCH - 1].after(sentinel);

  clientSidebarObserver = new IntersectionObserver(function(entries) {
    if (entries[0].isIntersecting) {
      loadMoreClientSidebarItems(list);
    }
  }, { root: list, rootMargin: '200px' });
  clientSidebarObserver.observe(sentinel);
}

function loadMoreClientSidebarItems(list) {
  var sentinel = document.getElementById('client-sidebar-sentinel');
  if (!sentinel) return;
  var batch = 0;
  var sibling = sentinel.nextElementSibling;
  while (sibling && batch < CLIENT_SIDEBAR_BATCH) {
    var next = sibling.nextElementSibling;
    if (sibling.hasAttribute('data-sidebar-hidden')) {
      sibling.removeAttribute('data-sidebar-hidden');
      batch++;
      if (batch >= CLIENT_SIDEBAR_BATCH) {
        sibling.after(sentinel);
        break;
      }
    }
    sibling = next;
  }
  if (!list.querySelector('[data-sidebar-hidden]')) {
    if (clientSidebarObserver) { clientSidebarObserver.disconnect(); clientSidebarObserver = null; }
    if (sentinel) sentinel.remove();
  }
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
      icon.innerHTML = tray.classList.contains('hidden') ? heroicon("paper-clip") : heroicon("x-mark");
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
    if (icon) icon.innerHTML = heroicon("paper-clip");
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
      icon.innerHTML = heroicon("paper-clip");
    }
  }
});




