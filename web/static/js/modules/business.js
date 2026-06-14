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
  if (!clientName) {
    var el = document.querySelector('[data-client-id="' + clientId + '"]');
    clientName = el ? el.getAttribute('data-client-name') || 'this client' : 'this client';
  }
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

  // Long-press on client list items (touch)
  var clientList = document.getElementById('client-list');
  if (clientList) {
    var longTimer = null;
    clientList.addEventListener('touchstart', function(e) {
      var item = e.target.closest('.wa-chat-item');
      if (item && e.touches.length === 1) {
        var id = item.getAttribute('data-client-id');
        if (id) {
          longTimer = setTimeout(function() {
            waShowCtxMenu({clientX: e.touches[0].clientX, clientY: e.touches[0].clientY, preventDefault: function(){}, stopPropagation: function(){}}, id);
          }, 500);
        }
      }
    });
    clientList.addEventListener('touchend', function() { clearTimeout(longTimer); });
    clientList.addEventListener('touchmove', function() { clearTimeout(longTimer); });
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
    if (item && !e.target.closest('.conversation-progress-dropdown') && !e.target.closest('.save-progress-btn') && !e.target.closest('.fa-trash-alt')) {
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

// === Notification Context Menu & Delete ===
let notifCtxTarget = null, notifCtxId = null;

function deleteNotification(id, btn) {
  showConfirmModal({
    title: 'Delete Notification',
    message: 'Delete this notification?',
    confirmText: 'Delete',
    confirmClass: 'bg-[var(--color-error)] text-white'
  }).then(function(confirmed) {
    if (!confirmed) return;
    btn.disabled = true;
    var row = btn.closest('[data-notif-id]') || btn;
    fetch('/business/notifications/' + id, {
      method: 'DELETE',
      headers: { 'X-CSRF-Token': getCookie('csrf_token') }
    }).then(function(r) { return r.json(); })
      .then(function(d) {
        if (d.success) {
          row.style.opacity = 0;
          setTimeout(function() { row.remove(); }, 160);
          showNotification('Notification deleted', 'success');
          var countEl = document.querySelector('.text-xs.text-[var(--color-text-muted)]');
          if (countEl && countEl.textContent.match(/^\d+ unread/)) {
            var n = parseInt(countEl.textContent); n = Math.max(n-1,0);
            countEl.textContent = n === 0 ? 'All read' : n + ' unread';
          }
        } else {
          btn.disabled = false;
          showNotification(d.error || 'Failed to delete', 'error');
        }
      })
      .catch(function() {
        btn.disabled = false;
        showNotification('Failed to delete', 'error');
      });
  });
}

function hideNotifCtxMenu() {
  var m = document.getElementById('notifCtxMenu');
  if (m) m.classList.add('hidden');
  notifCtxTarget = notifCtxId = null;
}

function showNotifCtxMenu(e, notifId, notifRow) {
  var m = document.getElementById('notifCtxMenu');
  if (!m) return;
  notifCtxTarget = notifRow;
  notifCtxId = notifId;
  m.classList.remove('hidden');
  m.style.left = Math.min(e.clientX, window.innerWidth - 160) + 'px';
  m.style.top = Math.min(e.clientY, window.innerHeight - 100) + 'px';
  e.preventDefault();
}

// Event delegation for notification context menu (right-click + long-press)
// Uses document delegation because notification content is loaded via HTMX
document.addEventListener('contextmenu', function(e) {
  var row = e.target.closest('[data-notif-id]');
  if (row) {
    showNotifCtxMenu(e, row.getAttribute('data-notif-id'), row);
  }
});
// Long-press on notification items (touch)
(function() {
  var longTimer = null;
  document.addEventListener('touchstart', function(e) {
    if (e.touches.length === 1) {
      var row = e.target.closest('[data-notif-id]');
      if (row) {
        longTimer = setTimeout(function() {
          showNotifCtxMenu(e.touches[0], row.getAttribute('data-notif-id'), row);
        }, 450);
      }
    }
  });
  document.addEventListener('touchend', function() { clearTimeout(longTimer); });
  document.addEventListener('touchmove', function() { clearTimeout(longTimer); });
})();

// Context menu action buttons (document delegation — loaded dynamically via HTMX)
document.addEventListener('click', function(e) {
  if (e.target.closest('#notifMarkReadBtn')) {
    if (!notifCtxId) return hideNotifCtxMenu();
    fetch('/business/notifications/' + notifCtxId + '/read', {
      method: 'POST',
      headers: { 'X-CSRF-Token': getCookie('csrf_token') }
    }).then(function(r) { return r.json(); })
      .then(function() {
        if (notifCtxTarget) notifCtxTarget.classList.add('opacity-60');
        showNotification('Marked as read', 'success');
        hideNotifCtxMenu();
      });
  }
  if (e.target.closest('#notifDeleteBtn')) {
    if (!notifCtxId) return hideNotifCtxMenu();
    deleteNotification(notifCtxId, notifCtxTarget || document);
    hideNotifCtxMenu();
  }
});

// Hide menu on click-outside / escape
window.addEventListener('click', function(e) {
  var m = document.getElementById('notifCtxMenu');
  if (m && !m.classList.contains('hidden') && !m.contains(e.target)) hideNotifCtxMenu();
});
document.addEventListener('keydown', function(e) {
  if (e.key === 'Escape') hideNotifCtxMenu();
});
// === End Notification Context Menu ===
