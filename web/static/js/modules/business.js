let currentClientId = null;

function showNewClientModal() {
  document.getElementById('new-client-modal').classList.remove('hidden');
}

function hideNewClientModal() {
  document.getElementById('new-client-modal').classList.add('hidden');
  document.getElementById('new-client-form').reset();
}

function loadClient(clientId) {
  currentClientId = clientId;
  document.querySelectorAll('.client-item').forEach(function(item) {
    item.classList.remove('bg-[var(--color-info-light)]', 'border-l-4', 'border-[var(--color-info)]');
  });
  var el = document.querySelector('[data-client-id="' + clientId + '"]');
  if (el) el.classList.add('bg-[var(--color-info-light)]', 'border-l-4', 'border-[var(--color-info)]');
  htmx.ajax('GET', 'clients/' + clientId + '/messages', {
    target: '#chat-area',
    swap: 'innerHTML'
  });
  if (window.innerWidth < 1024) {
    var layout = document.getElementById('mainLayout');
    var overlay = document.getElementById('sidebarOverlay');
    if (layout) layout.classList.remove('sidebar-open');
    if (overlay) overlay.classList.add('hidden');
  }
}

function deleteClient(clientId, clientName) {
  if (!confirm('Are you sure you want to delete "' + clientName + '"? This action cannot be undone.')) return;
  fetch('clients/' + clientId, { method: 'DELETE', headers: { 'Content-Type': 'application/json' } })
    .then(function(r) { return r.json(); })
    .then(function(data) {
      if (data.success) {
        showNotification('Customer deleted successfully!', 'success');
        var el = document.querySelector('[data-client-id="' + clientId + '"]');
        if (el) el.remove();
        if (currentClientId == clientId) {
          document.getElementById('chat-area').innerHTML =
            '<div class="flex-1 flex items-center justify-center text-[var(--color-text-muted)]">' +
            '<div class="text-center text-[var(--color-text-muted)]">' +
            '<i class="fas fa-comments text-6xl mb-4"></i>' +
            '<p>Select a client to start chatting</p></div></div>';
          currentClientId = null;
        }
      } else {
        showNotification(data.error || 'Failed to delete client', 'error');
      }
    })
    .catch(function() { showNotification('Failed to delete client', 'error'); });
}

// Client search filter
function filterClients() {
  var q = document.getElementById('clientSearch').value.toLowerCase().trim();
  document.querySelectorAll('.client-item').forEach(function(el) {
    var name = el.querySelector('h3')?.textContent?.toLowerCase() || '';
    var email = el.querySelector('.text-slate-400')?.textContent?.toLowerCase() || '';
    el.style.display = (!q || name.includes(q) || email.includes(q)) ? '' : 'none';
  });
}

document.addEventListener('DOMContentLoaded', function() {
  // New client form
  var form = document.getElementById('new-client-form');
  if (form) {
    form.addEventListener('submit', function(e) {
      e.preventDefault();
      fetch('clients', { method: 'POST', body: new FormData(this) })
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

  // Click delegation for client items and progress buttons
  document.addEventListener('click', function(e) {
    var saveBtn = e.target.closest('.save-progress-btn');
    if (saveBtn) {
      var id = saveBtn.getAttribute('data-customer-id');
      var dd = document.querySelector('.conversation-progress-dropdown[data-customer-id="' + id + '"]');
      if (dd && dd.value) saveConversationProgress(id, dd.value);
    }
    var item = e.target.closest('.client-item');
    if (item && !e.target.closest('.conversation-progress-dropdown') && !e.target.closest('.save-progress-btn')) {
      loadClient(item.getAttribute('data-client-id'));
    }
  });
});

function openPaymentModal(clientId) {
  var conversationId = document.querySelector('[data-client-id="' + clientId + '"]')?.closest?.('[data-conversation-id]');
  // Implement payment modal via the existing RequestPayment endpoint
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

function toggleStats() {
  var grid = document.getElementById('statsGrid');
  var btn = document.querySelector('.stats-toggle');
  if (grid) grid.classList.toggle('expanded');
  if (btn) btn.classList.toggle('expanded');
}

function toggleActionButtons() {
  var buttons = document.getElementById('action-buttons');
  var icon = document.getElementById('action-icon');
  var mainBtn = document.getElementById('main-action-btn');
  if (buttons.classList.contains('hidden')) {
    buttons.classList.remove('hidden');
    icon.classList.replace('fa-plus', 'fa-times');
    if (mainBtn) mainBtn.classList.add('rotate-45');
    buttons.querySelectorAll('button').forEach(function(btn, i) {
      btn.style.cssText = 'opacity:0;transform:translateY(20px)';
      setTimeout(function() { btn.style.cssText = 'opacity:1;transform:translateY(0)'; }, i * 50);
    });
  } else {
    buttons.classList.add('hidden');
    icon.classList.replace('fa-times', 'fa-plus');
    if (mainBtn) mainBtn.classList.remove('rotate-45');
  }
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
      fetch('/conversations/' + data.conversation_id + '/stage', { method: 'PUT', body: fd })
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
    icon.classList.toggle('fa-paperclip');
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
    icon.classList.replace('fa-times', 'fa-paperclip');
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
