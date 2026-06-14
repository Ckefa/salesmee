let pollingInterval = null;

scrollToBottom();
markAsRead();
startMessagePolling();

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

function startMessagePolling() {
  pollingInterval = setInterval(function() {
    fetchMessages();
  }, 5000);
}

function fetchMessages() {
  fetch(`/business/clients/${clientId}/messages`)
    .then(response => response.text())
    .then(html => {
      const parser = new DOMParser();
      const doc = parser.parseFromString(html, 'text/html');
      const newMessages = doc.getElementById('messages-container');
      const currentMessages = document.getElementById('messages-container');

      if (newMessages && currentMessages && newMessages.innerHTML !== currentMessages.innerHTML) {
        currentMessages.innerHTML = newMessages.innerHTML;
        currentMessages.scrollTop = currentMessages.scrollHeight;
        markAsRead();
        playNotificationSound();
      }
    })
    .catch(error => {
      console.error('Error fetching messages:', error);
    });
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
        fetchMessages();
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
  if (pollingInterval) {
    clearInterval(pollingInterval);
  }
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
    drawer.classList.add('open');
    if (!drawer.hasChildNodes() || drawer.innerHTML.trim() === '') {
      drawer.innerHTML = '<div class="px-3 sm:px-6 py-6 text-center text-[var(--color-text-muted)] text-sm"><i class="fas fa-spinner fa-spin mr-2"></i>Loading insights...</div>';
      htmx.ajax('GET', '/business/conversations/' + conversationId + '/insights-panel', {
        target: '#insights-drawer',
        swap: 'innerHTML'
      });
    }
  } else {
    drawer.classList.remove('open');
  }
}

// ========== Order Lifecycle Functions ==========

function sendOrderToClient(orderId) {
  fetch(`/business/orders/${orderId}/send`, { method: 'POST', headers: { 'X-CSRF-Token': getCookie('csrf_token') } })
    .then(r => r.json())
    .then(data => {
      if (data.success) {
        showNotification('Order sent to client!', 'success');
        fetchMessages();
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
        fetchMessages();
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
        fetchMessages();
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
        fetchMessages();
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
        fetchMessages();
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
        fetchMessages();
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
        fetchMessages();
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
}

function onMessageKeydown(event) {
  var qr = document.getElementById('quickReplies');
  if (event.key === 'Escape' && qr && !qr.classList.contains('hidden')) {
    qr.classList.add('hidden');
    var input = document.getElementById('messageInput');
    if (input) input.value = input.value.replace(/\/$/, '');
  }
}




