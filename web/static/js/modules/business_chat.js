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

function updateClientStatus(clientId, status) {
  console.log('Updating client status:', clientId, status);
  const formData = new FormData();
  formData.append('status', status);

  fetch('/business/clients/' + clientId + '/status', {
    method: 'PUT',
    headers: { 'X-CSRF-Token': getCookie('csrf_token') },
    body: formData
  })
    .then(response => {
      console.log('Customer status response status:', response.status);
      if (!response.ok) {
        throw new Error('HTTP error! status: ' + response.status);
      }
      return response.json();
    })
    .then(data => {
      console.log('Customer status response data:', data);
      if (data.client) {
        showNotification('Customer status updated to ' + status, 'success');
        const dropdown = document.querySelector('select[data-client-id="' + clientId + '"]');
        if (dropdown) {
          dropdown.value = status;
          console.log('Updated customer dropdown to:', status);
        }
      } else {
        console.error('No client data in response');
        showNotification('Failed to update client status', 'error');
      }
    })
    .catch(error => {
      console.error('Error updating client status:', error);
      showNotification('Failed to update client status: ' + error.message, 'error');
    });
}

function showEnhancedActionModal(messageId) {
  htmx.ajax('GET', '/actions/modal/' + messageId, {
    target: 'body',
    swap: 'beforeend'
  });
}

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

function createQuickAction(messageId, actionType) {
  const formData = new FormData();
  formData.append('message_id', messageId);
  formData.append('type', actionType);
  formData.append('title', actionType.charAt(0).toUpperCase() + actionType.slice(1) + ' from message');

  htmx.ajax('POST', 'business/messages/' + messageId + '/actions', {
    values: formData,
    target: '#action-result',
    swap: 'innerHTML'
  });
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

function orderItemIncrement(orderId, productId, btn) {
  fetch(`/business/orders/${orderId}/items/${productId}/increment`, { method: 'POST', headers: { 'X-CSRF-Token': getCookie('csrf_token') } })
    .then(r => r.json())
    .then(data => {
      if (data.success) fetchMessages();
    })
    .catch(console.error);
}

function orderItemDecrement(orderId, productId, btn) {
  fetch(`/business/orders/${orderId}/items/${productId}/decrement`, { method: 'POST', headers: { 'X-CSRF-Token': getCookie('csrf_token') } })
    .then(r => r.json())
    .then(data => {
      if (data.success) fetchMessages();
    })
    .catch(console.error);
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

// ========== Message Search ==========

function toggleMessageSearch() {
  var bar = document.getElementById('messageSearchBar');
  if (bar) bar.classList.toggle('hidden');
  if (!bar.classList.contains('hidden')) {
    setTimeout(function() {
      document.getElementById('messageSearchInput')?.focus();
    }, 100);
  } else {
    clearMessageSearch();
  }
}

function filterMessages(query) {
  var q = query.toLowerCase().trim();
  var container = document.getElementById('messages-container');
  var messages = container.querySelectorAll(':scope > div');
  var count = 0;
  messages.forEach(function(el) {
    var text = el.getAttribute('data-message-text') || el.textContent.toLowerCase();
    if (!q || text.toLowerCase().includes(q)) {
      el.style.display = '';
      count++;
    } else {
      el.style.display = 'none';
    }
  });
  var countEl = document.getElementById('searchResultCount');
  if (countEl) countEl.textContent = count;
}

function clearMessageSearch() {
  var input = document.getElementById('messageSearchInput');
  if (input) input.value = '';
  var container = document.getElementById('messages-container');
  if (container) {
    container.querySelectorAll(':scope > div').forEach(function(el) {
      el.style.display = '';
    });
  }
  var countEl = document.getElementById('searchResultCount');
  if (countEl) countEl.textContent = '0';
}

// ========== Customer Info Panel ==========

function toggleCustomerInfo() {
  var panel = document.getElementById('customerInfoPanel');
  if (panel) panel.classList.toggle('hidden');
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

function insertQuickReply(text) {
  var input = document.getElementById('messageInput');
  if (input) {
    input.value = text;
    input.focus();
  }
  var qr = document.getElementById('quickReplies');
  if (qr) qr.classList.add('hidden');
}


