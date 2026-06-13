let currentClientId = null;
let ctxMenuClientId = null;

function waShowCtxMenu(e, clientId) {
  e.preventDefault();
  e.stopPropagation();
  ctxMenuClientId = clientId;
  var menu = document.getElementById('ctxMenu');
  if (!menu) return;

  var x = e.clientX;
  var y = e.clientY;
  var w = window.innerWidth;
  var h = window.innerHeight;
  var mw = 200;
  var mh = menu.offsetHeight || 128;
  if (x + mw > w) x = w - mw - 8;
  if (y + mh > h) y = h - mh - 8;
  if (x < 8) x = 8;
  if (y < 8) y = 8;

  menu.style.left = x + 'px';
  menu.style.top = y + 'px';
  menu.classList.remove('hidden');
}

function waHideCtxMenu() {
  var menu = document.getElementById('ctxMenu');
  if (menu) menu.classList.add('hidden');
  ctxMenuClientId = null;
}

function waCtxMarkRead() {
  if (!ctxMenuClientId) return;
  fetch('clients/' + ctxMenuClientId + '/read', {
    method: 'PUT',
    headers: { 'X-CSRF-Token': getCookie('csrf_token') }
  }).then(function(r) { return r.json(); })
    .then(function(d) {
      if (d.status === 'ok') {
        showNotification('Marked as read', 'success');
        var el = document.querySelector('[data-client-id="' + ctxMenuClientId + '"]');
        if (el) {
          el.setAttribute('data-unread', '0');
          var badge = el.querySelector('.wa-unread-badge');
          if (badge) badge.remove();
        }
      }
    })
    .catch(function() { showNotification('Failed to mark as read', 'error'); })
    .finally(function() { waHideCtxMenu(); });
}

function waCtxClearChat() {
  if (!ctxMenuClientId) return;
  if (!confirm('Clear all messages in this chat? This cannot be undone.')) { waHideCtxMenu(); return; }
  fetch('clients/' + ctxMenuClientId + '/messages', {
    method: 'DELETE',
    headers: { 'X-CSRF-Token': getCookie('csrf_token') }
  }).then(function(r) { return r.json(); })
    .then(function(d) {
      if (d.success) {
        showNotification('Chat cleared', 'success');
        if (currentClientId == ctxMenuClientId) {
          htmx.ajax('GET', 'clients/' + ctxMenuClientId + '/messages', {
            target: '#chat-area',
            swap: 'innerHTML'
          });
        }
      } else {
        showNotification(d.error || 'Failed to clear chat', 'error');
      }
    })
    .catch(function() { showNotification('Failed to clear chat', 'error'); })
    .finally(function() { waHideCtxMenu(); });
}

function waCtxDeleteChat() {
  if (!ctxMenuClientId) return;
  var el = document.querySelector('[data-client-id="' + ctxMenuClientId + '"]');
  var name = el ? el.getAttribute('data-client-name') : 'this client';
  deleteClient(ctxMenuClientId, name);
  waHideCtxMenu();
}

function showNewClientModal() {
  document.getElementById('new-client-modal').classList.remove('hidden');
}

function hideNewClientModal() {
  document.getElementById('new-client-modal').classList.add('hidden');
  document.getElementById('new-client-form').reset();
}

function loadClient(clientId) {
  currentClientId = clientId;
  var layout = document.getElementById('mainLayout');

  document.querySelectorAll('.wa-chat-item').forEach(function(item) {
    item.classList.remove('selected');
  });
  var el = document.querySelector('[data-client-id="' + clientId + '"]');
  if (el) el.classList.add('selected');

  htmx.ajax('GET', 'clients/' + clientId + '/messages', {
    target: '#chat-area',
    swap: 'innerHTML'
  });

  if (window.innerWidth < 1024) {
    layout.classList.add('wa-chat-open');
  }
}

function waBackToChatList() {
  var layout = document.getElementById('mainLayout');
  layout.classList.remove('wa-chat-open');
}

function deleteClient(clientId, clientName) {
  if (!confirm('Are you sure you want to delete "' + clientName + '"? This action cannot be undone.')) return;
  fetch('clients/' + clientId, { method: 'DELETE', headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': getCookie('csrf_token') } })
    .then(function(r) { return r.json(); })
    .then(function(data) {
      if (data.success) {
        showNotification('Customer deleted successfully!', 'success');
        var el = document.querySelector('[data-client-id="' + clientId + '"]');
        if (el) el.remove();
        if (currentClientId == clientId) {
          var chatArea = document.getElementById('chat-area');
          if (chatArea) {
            chatArea.innerHTML =
              '<div class="wa-empty-state">' +
              '<img src="/static/images/salesmeebrand.png" class="wa-empty-state-logo">' +
              '<h2 class="wa-empty-state-title">SalesMee</h2>' +
              '<p class="wa-empty-state-text">Send and receive messages, Track orders, bookings, and payments from clients in one Platform.</p>' +
              '</div>';
          }
          currentClientId = null;
          waBackToChatList();
        }
      } else {
        showNotification(data.error || 'Failed to delete client', 'error');
      }
    })
    .catch(function() { showNotification('Failed to delete client', 'error'); });
}

function filterClients() {
  var q = document.getElementById('clientSearch').value.toLowerCase().trim();
  document.querySelectorAll('.wa-chat-item').forEach(function(el) {
    var name = el.getAttribute('data-client-name')?.toLowerCase() || '';
    var email = el.getAttribute('data-client-email')?.toLowerCase() || '';
    var preview = el.querySelector('.wa-chat-preview')?.textContent?.toLowerCase() || '';
    el.style.display = (!q || name.includes(q) || email.includes(q) || preview.includes(q)) ? '' : 'none';
  });
}

document.addEventListener('DOMContentLoaded', function() {
  var form = document.getElementById('new-client-form');
  if (form) {
    form.addEventListener('submit', function(e) {
      e.preventDefault();
      fetch('clients', { method: 'POST', headers: { 'X-CSRF-Token': getCookie('csrf_token') }, body: new FormData(this) })
        .then(function(r) { return r.json(); })
        .then(function(data) {
          if (data.success) {
            hideNewClientModal();
            showNotification('Client added successfully!', 'success');
            setTimeout(function() { window.location.href = '/business'; }, 1500);
          } else {
            showNotification(data.error || 'Failed to add client', 'error');
          }
        })
        .catch(function() { showNotification('Failed to add client', 'error'); });
    });
  }

  document.addEventListener('click', function(e) {
    var ctxMenu = document.getElementById('ctxMenu');
    if (ctxMenu && !ctxMenu.classList.contains('hidden') && !ctxMenu.contains(e.target)) {
      waHideCtxMenu();
    }
    var saveBtn = e.target.closest('.save-progress-btn');
    if (saveBtn) {
      var id = saveBtn.getAttribute('data-customer-id');
      var dd = document.querySelector('.conversation-progress-dropdown[data-customer-id="' + id + '"]');
      if (dd && dd.value) saveConversationProgress(id, dd.value);
    }
    var item = e.target.closest('.wa-chat-item');
    if (item && !e.target.closest('.conversation-progress-dropdown') && !e.target.closest('.save-progress-btn')) {
      loadClient(item.getAttribute('data-client-id'));
    }
  });

  document.addEventListener('contextmenu', function(e) {
    if (!e.target.closest('.wa-chat-item') && !e.target.closest('#ctxMenu')) {
      waHideCtxMenu();
    }
  });

  var assistContainer = document.getElementById('assist-overlay-container');
  if (assistContainer && assistContainer.querySelector('.assist-panel')) {
    document.addEventListener('click', function(e) {
      if (e.target.closest('.assist-backdrop') || e.target.closest('.assist-close')) {
        assistContainer.classList.add('hidden');
      }
    });
  }
});

function openPaymentModal(clientId) {
  htmx.ajax('GET', '/business/clients/' + clientId + '/request-payment', {
    target: '#payment-modal',
    swap: 'innerHTML'
  });
  var modal = document.getElementById('payment-modal');
  if (!modal) {
    modal = document.createElement('div');
    modal.id = 'payment-modal';
    modal.className = 'fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50';
    document.body.appendChild(modal);
  }
  modal.classList.remove('hidden');
}

function sendMessage() { var form = document.getElementById('message-form'); if (form) form.submit(); }

function createMessageAction(messageId, type) {
  var title = prompt('Create ' + type + ':');
  if (!title) return;
  var description = prompt('Description (optional):') || '';
  var dueDate = type === 'booking' ? prompt('Date (YYYY-MM-DD):') : null;
  var fd = new FormData();
  fd.append('type', type);
  fd.append('title', title);
  fd.append('description', description);
  if (dueDate) fd.append('due_date', dueDate);
  htmx.ajax('POST', '/messages/' + messageId + '/actions', {
    target: '#actions-panel', swap: 'innerHTML', values: fd
  });
}

function saveConversationProgress(clientId, stage) {
  fetch('clients/' + clientId + '/conversation-id')
    .then(function(r) { return r.json(); })
    .then(function(data) {
      if (!data.conversation_id) { showNotification('Failed to get conversation ID', 'error'); return; }
      var fd = new FormData();
      fd.append('current_stage', stage);
      fd.append('progress_score', getProgressScore(stage));
      fetch('/conversations/' + data.conversation_id + '/stage', { method: 'PUT', headers: { 'X-CSRF-Token': getCookie('csrf_token') }, body: fd })
        .then(function(r) { return r.ok ? showNotification('Conversation progress updated!', 'success') : showNotification('Failed to update progress', 'error'); })
        .catch(function() { showNotification('Failed to save conversation progress', 'error'); });
    })
    .catch(function() { showNotification('Failed to get conversation information', 'error'); });
}

function toggleMediaTray() {
  var tray = document.getElementById('media-tray');
  var icon = document.getElementById('media-icon');
  if (tray) {
    tray.classList.toggle('hidden');
    icon.classList.toggle('fa-plus');
    icon.classList.toggle('fa-times');
  }
}

function triggerMediaUpload(type) {
  var input = document.getElementById('media-input-' + type);
  if (input) input.click();
  var tray = document.getElementById('media-tray');
  if (tray && !tray.classList.contains('hidden')) {
    tray.classList.add('hidden');
    var icon = document.getElementById('media-icon');
    icon.classList.replace('fa-times', 'fa-plus');
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
      icon.classList.replace('fa-times', 'fa-plus');
    }
  }
});
