// Shared functions from chat_common.js

function updateSidebarCard(frame) {
  if (frame.sender_type !== 'client') return;
  var msg = frame.new_message;
  if (!msg) return;
  if (msg.msg_type === 'order' || msg.msg_type === 'booking') return;

  var cid = frame.sender_id;
  var item = document.querySelector('.wa-chat-item[data-client-id="' + cid + '"]');
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

  if (msg.created_at) {
    item.setAttribute('data-last-message-at', new Date(Number(msg.created_at)).toISOString());
  }

  // Increment unread badge
  var badge = item.querySelector('.wa-unread-badge');
  if (badge) {
    var count = parseInt(badge.textContent) + 1;
    badge.textContent = count > 99 ? '99+' : count;
  } else {
    var topRight = item.querySelector('.wa-chat-top-right');
    if (topRight) {
      topRight.insertAdjacentHTML('beforeend', '<span class="wa-unread-badge">1</span>');
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
      }
    })
    .catch(console.error);
}

function updateOrderPendingPaymentsUI(orderId, pendingAmount) {
  var el = document.getElementById('orderPendingPayments-' + orderId);
  if (!el) return;
  if (pendingAmount > 0) {
    el.innerHTML =
      '<div class="text-[10px] font-medium text-[var(--color-warning)] mb-2"><i class="fas fa-clock mr-0.5"></i>Awaiting payment confirmation</div>' +
      '<button class="w-full py-1.5 px-3 rounded-lg bg-[var(--color-success)] text-white hover:opacity-90 text-xs font-medium transition shadow-sm" onclick="confirmAllOrderPayments(' + orderId + ')">' +
      '<i class="fas fa-check mr-1"></i>Confirm Payment</button>';
  } else {
    el.innerHTML = '<div class="text-[10px] font-medium text-[var(--color-text-muted)]"><i class="fas fa-clock mr-0.5"></i>No pending payment claims</div>';
  }
}

function updateBookingPendingPaymentsUI(bookingId, pendingAmount) {
  var el = document.getElementById('bookingPendingPayments-' + bookingId);
  if (!el) return;
  if (pendingAmount > 0) {
    el.innerHTML =
      '<div class="text-[10px] font-medium text-[var(--color-warning)] mb-2"><i class="fas fa-clock mr-0.5"></i>Awaiting payment confirmation</div>' +
      '<button class="w-full py-1.5 px-3 rounded-lg bg-[var(--color-success)] text-white hover:opacity-90 text-xs font-medium transition shadow-sm" onclick="confirmAllBookingPayments(' + bookingId + ')">' +
      '<i class="fas fa-check mr-1"></i>Confirm Payment</button>';
  } else {
    el.innerHTML = '<div class="text-[10px] font-medium text-[var(--color-text-muted)]"><i class="fas fa-clock mr-0.5"></i>No pending payment claims</div>';
  }
}

function applyOrderCardUpdate(upd) {
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

  var patched = false;
  if (typeof upd.pending_amount === 'number') {
    updateOrderPendingPaymentsUI(upd.order_id, upd.pending_amount);
    patched = true;
  }
  if (upd.has_review) {
    addReviewBadgeToCard(card, upd.review_rating || 5);
    patched = true;
  }
  return patched;
}

function applyBookingCardUpdate(upd) {
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

  var patched = false;
  if (typeof upd.pending_amount === 'number') {
    updateBookingPendingPaymentsUI(upd.booking_id, upd.pending_amount);
    patched = true;
  }
  if (upd.has_review) {
    addReviewBadgeToCard(card, upd.review_rating || 5);
    patched = true;
  }
  return patched;
}

function wsToken() {
  return window.AUTH_TOKEN || getCookie('token') || getCookie('team_token') || '';
}

function startWsClient() {
  if (!window.wsClient) {
    window.wsClient = new WsClient();
    var token = wsToken();
    if (!token) return;
    window.wsClient.connect('/ws/business?token=' + encodeURIComponent(token) + '&business_id=' + encodeURIComponent(window.BUSINESS_ID || ''));
  }
  registerChatHandlers();
}

function registerChatHandlers() {
  if (window._chatHandlersRegistered) return;
  window._chatHandlersRegistered = true;

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
    if (frame.sender_type === 'business') return;
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
    playNotificationSound();
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
    if (!applyOrderCardUpdate(upd)) {}
  });

  window.wsClient.on(7, function(frame) {
    var upd = frame.booking_update;
    if (!upd) return;
    if (!applyBookingCardUpdate(upd)) {}
  });

  window.wsClient.on(8, function(frame) {
    if (!frame.unread_count) return;
    var uc = frame.unread_count;
    if (!uc.conversation_id) return;
    var item = document.querySelector('.wa-chat-item[data-conversation-id="' + uc.conversation_id + '"]');
    if (!item) return;
    var badge = item.querySelector('.wa-unread-badge');
    if (uc.count > 0) {
      if (badge) {
        badge.textContent = uc.count > 99 ? '99+' : uc.count;
      } else {
        var topRight = item.querySelector('.wa-chat-top-right');
        if (topRight) {
          topRight.insertAdjacentHTML('beforeend', '<span class="wa-unread-badge">' + (uc.count > 99 ? '99+' : uc.count) + '</span>');
        }
      }
    } else {
      if (badge) badge.remove();
    }
  });

  window.wsClient.on(2, function(frame) {
    applyReadReceipt(frame.read_receipt);
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

function playNotificationSound() {
  try {
    var audio = new Audio('/static/sounds/notification.mp3');
    audio.volume = 0.3;
    audio.play();
  } catch(e) {}
}

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
      } else {
        showNotification(data.error || `Failed to ${action} booking`, 'error');
      }
    })
    .catch(e => { console.error(e); showNotification(`Failed to ${action} booking`, 'error'); });
  });
}


