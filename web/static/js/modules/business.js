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
  showConfirmModal({ title: 'Clear Chat', message: 'Clear all messages in this chat? This cannot be undone.', confirmText: 'Clear', confirmClass: 'bg-[var(--color-error)] text-white' }).then(function(confirmed) {
    if (!confirmed) { waHideCtxMenu(); return; }
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
  });
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

function buildSkeletonChatContainer(clientId) {
  var el = document.querySelector('[data-client-id="' + clientId + '"]');
  var name = el ? el.getAttribute('data-client-name') || 'Loading...' : 'Loading...';
  var convId = window.conversationId || '';
  return '<div class="wa-chat-container" id="waChatContainer">' +
    '<div class="wa-chat-header">' +
      '<button onclick="waBackToChatList()" class="wa-chat-back" title="Back to chats" id="chatBackBtn">' +
        heroicon("arrow-left") +
      '</button>' +
      '<div class="wa-chat-header-avatar">' +
        '<div class="avatar avatar-placeholder avatar-sm">' + heroicon("user", "text-white", "text-xs") + '</div>' +
      '</div>' +
      '<div class="wa-chat-header-info">' +
        '<h2 class="wa-chat-header-name">' + escapeHtml(name) + '</h2>' +
        '<span class="wa-chat-header-status"><span class="wa-header-online-text">loading...</span></span>' +
      '</div>' +
      '<div class="wa-chat-header-actions relative">' +
        '<div class="skeleton rounded-lg w-8 h-8"></div>' +
        '<div class="skeleton rounded-lg w-8 h-8 ml-1"></div>' +
      '</div>' +
    '</div>' +
    '<div class="wa-progress-bar">' +
      '<div class="border-b border-[var(--color-border)] px-3 sm:px-6 py-1.5">' +
        '<div class="flex items-center justify-between gap-1.5 text-xs leading-tight">' +
          '<div class="skeleton rounded h-3 w-32"></div>' +
          '<div class="skeleton rounded h-3 w-24"></div>' +
        '</div>' +
      '</div>' +
    '</div>' +
    '<div id="insights-drawer"></div>' +
    '<div class="wa-messages-area" id="messages-container">' +
      '<div class="flex-1 p-4 space-y-4">' +
        '<div class="skeleton skeleton-card"></div>' +
        '<div class="flex justify-end"><div class="skeleton skeleton-card" style="width:60%"></div></div>' +
        '<div class="flex"><div class="skeleton skeleton-card" style="width:70%"></div></div>' +
        '<div class="flex justify-end"><div class="skeleton skeleton-card" style="width:45%"></div></div>' +
        '<div class="flex"><div class="skeleton skeleton-card" style="width:55%"></div></div>' +
        '<div class="flex justify-end"><div class="skeleton skeleton-card" style="width:65%"></div></div>' +
      '</div>' +
    '</div>' +
    '<button class="wa-scroll-bottom" id="scrollToBottom" onclick="scrollToBottomBtn()" style="display:none">' +
      '<svg viewBox="0 0 24 24" height="24" width="24" fill="none"><path d="M11 13.6L6.11253 8.71253C5.72003 8.32003 5.08281 8.32285 4.69381 8.7188C4.30964 9.10983 4.31241 9.73741 4.70003 10.125L11.2669 16.6919C11.6718 17.0968 12.3282 17.0968 12.7331 16.6919L19.3 10.125C19.6876 9.73741 19.6904 9.10983 19.3062 8.7188C18.9172 8.32285 18.28 8.32003 17.8875 8.71253L13 13.6L12 14.625L11 13.6Z" fill="currentColor"/></svg>' +
      '<span class="scroll-bottom-badge" id="scrollBottomBadge"></span>' +
    '</button>' +
    '<div class="wa-input-wrapper">' +
      '<div class="wa-input-inner">' +
        '<div class="skeleton rounded-lg w-10 h-10"></div>' +
        '<div class="skeleton rounded-full h-10 flex-1"></div>' +
        '<div class="skeleton rounded-lg w-10 h-10"></div>' +
      '</div>' +
    '</div>' +
  '</div>';
}

function loadClient(clientId) {
  currentClientId = clientId;
  window.clientId = clientId;
  window.sender = 'business';
  var layout = document.getElementById('mainLayout');
  window.businessId = window.BUSINESS_ID;

  document.querySelectorAll('.wa-chat-item').forEach(function(item) {
    item.classList.remove('selected');
  });
  var el = document.querySelector('[data-client-id="' + clientId + '"]');
  if (el) {
    el.classList.add('selected');
    window.conversationId = el.getAttribute('data-conversation-id');
  }

  // Build chat container with real header + skeleton messages
  var contentArea = document.getElementById('content-area');
  contentArea.innerHTML = buildSkeletonChatContainer(clientId);
  contentArea.classList.remove('hidden');
  contentArea.classList.add('content-fade-in');

  var loadTimeout = setTimeout(function() {
    var mc = document.getElementById('messages-container');
    if (mc && mc.querySelector('.skeleton')) {
      mc.innerHTML = '<div class="flex flex-col items-center justify-center py-12 text-center flex-1">' +
        '<div class="w-14 h-14 rounded-full bg-[var(--color-error-light)] flex items-center justify-center mb-4">' +
        heroicon("exclamation-triangle", "text-[var(--color-error)]", "text-2xl") + '</div>' +
        '<p class="text-sm font-medium text-[var(--color-text)] mb-1">Failed to load</p>' +
        '<p class="text-xs text-[var(--color-text-muted)] mb-4">Timed out. Please try again.</p>' +
        '<button onclick="loadClient(' + clientId + ')" class="px-4 py-2 bg-[var(--color-primary)] text-white rounded-lg text-sm hover:opacity-90 transition-colors">' +
        heroicon("arrow-path", "mr-1") + ' Retry</button></div>';
    }
  }, 20000);

  fetch('clients/' + clientId + '/messages')
    .then(function(r) { return r.text(); })
    .then(function(html) {
      clearTimeout(loadTimeout);
      // Extract fragments from fetched HTML: messages-container, scroll button, input wrapper
      var parser = new DOMParser();
      var doc = parser.parseFromString(html, 'text/html');
      var newMessages = doc.getElementById('messages-container');
      var newScrollBtn = doc.getElementById('scrollToBottom');
      var newInput = doc.querySelector('.wa-input-wrapper');
      if (newMessages) {
        var oldMessages = document.getElementById('messages-container');
        if (oldMessages) oldMessages.outerHTML = newMessages.outerHTML;
      }
      if (newScrollBtn) {
        var oldBtn = document.getElementById('scrollToBottom');
        if (oldBtn) oldBtn.outerHTML = newScrollBtn.outerHTML;
      }
      if (newInput) {
        var oldInput = document.querySelector('.wa-input-wrapper');
        if (oldInput) oldInput.outerHTML = newInput.outerHTML;
      }
      // Replace header with fetched version for accurate info
      var newHeader = doc.querySelector('.wa-chat-header');
      if (newHeader) {
        var oldHeader = document.querySelector('.wa-chat-header');
        if (oldHeader) oldHeader.outerHTML = newHeader.outerHTML;
      }
      // Replace progress bar with fetched version
      var newProgress = doc.querySelector('.wa-progress-bar');
      if (newProgress) {
        var oldProgress = document.querySelector('.wa-progress-bar');
        if (oldProgress) oldProgress.outerHTML = newProgress.outerHTML;
      }
      htmx.process(contentArea);
      scrollToBottom();
      markAsRead();
      initOlderObserver();
      initScrollToBottom();
    })
    .catch(function() {
      clearTimeout(loadTimeout);
      var mc = document.getElementById('messages-container');
      if (mc) {
        mc.innerHTML = '<div class="flex flex-col items-center justify-center py-12 text-center flex-1">' +
          '<div class="w-14 h-14 rounded-full bg-[var(--color-error-light)] flex items-center justify-center mb-4">' +
          heroicon("exclamation-triangle", "text-[var(--color-error)]", "text-2xl") + '</div>' +
          '<p class="text-sm font-medium text-[var(--color-text)] mb-1">Failed to load</p>' +
          '<p class="text-xs text-[var(--color-text-muted)] mb-4">Something went wrong.</p>' +
          '<button onclick="loadClient(' + clientId + ')" class="px-4 py-2 bg-[var(--color-primary)] text-white rounded-lg text-sm hover:opacity-90 transition-colors">' +
          heroicon("arrow-path", "mr-1") + ' Retry</button></div>';
      }
      showNotification('Failed to load conversation', 'error');
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
  showConfirmModal({ title: 'Delete Customer', message: 'Are you sure you want to delete "' + clientName + '"? This action cannot be undone.', confirmText: 'Delete', confirmClass: 'bg-[var(--color-error)] text-white' }).then(function(confirmed) {
    if (!confirmed) return;
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
  });
}

function filterClients() {
  var q = document.getElementById('clientSearch').value.toLowerCase().trim();
  document.querySelectorAll('.wa-chat-item').forEach(function(el) {
    var name = el.getAttribute('data-client-name')?.toLowerCase() || '';
    var email = el.getAttribute('data-client-email')?.toLowerCase() || '';
    var preview = el.querySelector('.wa-chat-preview')?.textContent?.toLowerCase() || '';
    var match = !q || name.includes(q) || email.includes(q) || preview.includes(q);
    el.style.display = match ? '' : 'none';
    // Reveal hidden items that match the filter
    if (match && el.hasAttribute('data-sidebar-hidden')) {
      el.removeAttribute('data-sidebar-hidden');
    }
  });
}

// Sidebar virtual scrolling — limit visible cards, load more via IntersectionObserver
var SIDEBAR_BATCH = 100;
var sidebarObserver = null;

function initSidebarVirtualScroll() {
  var list = document.getElementById('client-list');
  if (!list) return;
  if (sidebarObserver) { sidebarObserver.disconnect(); sidebarObserver = null; }

  // Remove any existing sentinel
  var old = document.getElementById('wa-sidebar-sentinel');
  if (old) old.remove();
  // Remove existing scroll-limit attributes
  list.querySelectorAll('.wa-chat-item').forEach(function(el) { el.removeAttribute('data-sidebar-hidden'); });

  var items = list.querySelectorAll('.wa-chat-item');
  if (items.length <= SIDEBAR_BATCH) return;

  // Hide excess items
  for (var i = SIDEBAR_BATCH; i < items.length; i++) {
    items[i].setAttribute('data-sidebar-hidden', 'true');
  }

  // Create sentinel as last visible item
  var sentinel = document.createElement('div');
  sentinel.id = 'wa-sidebar-sentinel';
  sentinel.style.height = '1px';
  items[SIDEBAR_BATCH - 1].after(sentinel);

  sidebarObserver = new IntersectionObserver(function(entries) {
    if (entries[0].isIntersecting) {
      loadMoreSidebarItems(list);
    }
  }, { root: list, rootMargin: '200px' });
  sidebarObserver.observe(sentinel);
}

function loadMoreSidebarItems(list) {
  var sentinel = document.getElementById('wa-sidebar-sentinel');
  if (!sentinel) return;
  var batch = 0;
  var sibling = sentinel.nextElementSibling;
  while (sibling && batch < SIDEBAR_BATCH) {
    var next = sibling.nextElementSibling;
    if (sibling.hasAttribute('data-sidebar-hidden')) {
      sibling.removeAttribute('data-sidebar-hidden');
      batch++;
      if (batch >= SIDEBAR_BATCH) {
        // Move sentinel after this batch
        sibling.after(sentinel);
        break;
      }
    }
    sibling = next;
  }
  // If no more hidden items, disconnect observer
  if (!list.querySelector('[data-sidebar-hidden]')) {
    if (sidebarObserver) { sidebarObserver.disconnect(); sidebarObserver = null; }
    if (sentinel) sentinel.remove();
  }
}

function autoSelectFirstChat() {
  if (window.clientId) return;
  var items = document.querySelectorAll('.wa-chat-item:not([data-sidebar-hidden])');
  if (!items.length) return;
  var target = null;
  for (var i = 0; i < items.length; i++) {
    if (parseInt(items[i].getAttribute('data-unread') || '0') > 0) {
      target = items[i];
      break;
    }
  }
  if (!target) target = items[0];
  var id = target.getAttribute('data-client-id');
  if (id) loadClient(id);
}

function registerBusinessPresenceHandlers() {
  if (window._businessPresenceRegistered) return;
  window._businessPresenceRegistered = true;
  window.wsClient.on(5, function(frame) {
    var p = frame.presence;
    if (!p) return;
    var el = document.querySelector('[data-client-id="' + p.client_id + '"] .wa-online-dot');
    if (el) {
      el.classList.remove('online', 'offline');
      el.classList.add(p.is_online ? 'online' : 'offline');
    }
    var card = document.querySelector('[data-client-id="' + p.client_id + '"]');
    if (card) {
      card.setAttribute('data-online', p.is_online ? 'true' : 'false');
      deferredSort();
    }
    if (String(p.client_id) === String(window.clientId || '')) {
      var statusEl = document.querySelector('.wa-header-online-text');
      if (statusEl) {
        statusEl.textContent = p.is_online ? 'online' : 'offline';
      }
    }
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
    item.setAttribute('data-unread', String(uc.count));
    deferredSort();
  });

  window.wsClient.on(14, function(frame) {
    if (!frame.pending_count) return;
    var pc = frame.pending_count;
    var updateBadge = function(id, count) {
      var el = document.getElementById(id);
      if (count > 0) {
        if (el) {
          el.textContent = count > 99 ? '99+' : count;
          el.style.display = '';
        } else {
          var parent = document.querySelector('[data-nav="' + (id === 'waOrdersBadge' ? 'orders' : id === 'waBookingsBadge' ? 'bookings' : 'notifications') + '"]');
          if (parent) {
            parent.insertAdjacentHTML('beforeend', '<span id="' + id + '" class="wa-nav-badge">' + (count > 99 ? '99+' : count) + '</span>');
          }
        }
      } else {
        if (el) el.style.display = 'none';
      }
    };
    updateBadge('waOrdersBadge', pc.order_count);
    updateBadge('waBookingsBadge', pc.booking_count);
    updateBadge('waNotifBadge', pc.notif_count);
  });
}

function startBusinessWS() {
  if (window.wsClient) {
    registerBusinessPresenceHandlers();
    return;
  }
  var token = window.AUTH_TOKEN || getCookie('token') || getCookie('team_token');
  if (!token) return;
  window.wsClient = new WsClient();
  window.wsClient.connect('/ws/business?token=' + encodeURIComponent(token) + '&business_id=' + (window.BUSINESS_ID || ''));
  registerBusinessPresenceHandlers();
}

window.addEventListener('beforeunload', function() {
  if (window.wsClient) window.wsClient.disconnect();
});

document.addEventListener('DOMContentLoaded', function() {
  startBusinessWS();
  sortClientList();
  initSidebarVirtualScroll();
  setTimeout(autoSelectFirstChat, 400);

  var form = document.getElementById('new-client-form');
  if (form) {
    form.addEventListener('submit', function(e) {
      e.preventDefault();
      var formEl = this;
      fetch('clients', { method: 'POST', headers: { 'X-CSRF-Token': getCookie('csrf_token') }, body: new FormData(this) })
        .then(function(r) { return r.json(); })
        .then(function(data) {
          if (data.success) {
            hideNewClientModal();
            showNotification('Client added successfully!', 'success');
            setTimeout(function() { window.location.href = '/business'; }, 1500);
          } else if (window.handlePlanResponse(data, function() {
            fetch('clients?grace=1', { method: 'POST', headers: { 'X-CSRF-Token': getCookie('csrf_token') }, body: new FormData(formEl) })
              .then(function(r) { return r.json(); })
              .then(function(d) {
                if (d.success) { hideNewClientModal(); showNotification('Client added successfully!', 'success'); setTimeout(function() { window.location.href = '/business'; }, 1500); }
                else { showNotification(d.error || 'Failed to add client', 'error'); }
              });
          })) {}
          else {
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
    if (item && !e.target.closest('.conversation-progress-dropdown') && !e.target.closest('.save-progress-btn') && !e.target.closest('.wa-chat-icon-btn')) {
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
  if (tray && icon) {
    tray.classList.toggle('hidden');
    icon.innerHTML = tray.classList.contains('hidden') ? heroicon("plus") : heroicon("x-mark");
  }
}

function triggerMediaUpload(type) {
  var input = document.getElementById('media-input-' + type);
  if (input) input.click();
  var tray = document.getElementById('media-tray');
  if (tray && !tray.classList.contains('hidden')) {
    tray.classList.add('hidden');
    var icon = document.getElementById('media-icon');
    if (icon) icon.innerHTML = heroicon("plus");
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
      icon.innerHTML = heroicon("plus");
    }
  }
});

// === WhatsApp-style pin & sort for client list ===
function togglePinClient(id) {
  var pins = JSON.parse(localStorage.getItem('pinned_clients') || '[]');
  var nid = parseInt(id);
  var idx = pins.indexOf(nid);
  if (idx > -1) { pins.splice(idx, 1); } else { pins.push(nid); }
  localStorage.setItem('pinned_clients', JSON.stringify(pins));
  sortClientList();
}

var _sortTimer = null;
function deferredSort() {
  if (_sortTimer) clearTimeout(_sortTimer);
  _sortTimer = setTimeout(sortClientList, 200);
}

function sortClientList() {
  var pins = JSON.parse(localStorage.getItem('pinned_clients') || '[]');
  var list = document.getElementById('client-list');
  if (!list) return;
  var sentinel = document.getElementById('wa-sidebar-sentinel');
  if (sentinel) sentinel.remove();
  // Reveal all hidden items so they participate in sorting
  list.querySelectorAll('[data-sidebar-hidden]').forEach(function(el) {
    el.removeAttribute('data-sidebar-hidden');
  });
  var items = Array.from(list.querySelectorAll('.wa-chat-item'));
  if (items.length < 2) return;
  items.sort(function(a, b) {
    var aP = pins.indexOf(parseInt(a.getAttribute('data-client-id'))) > -1;
    var bP = pins.indexOf(parseInt(b.getAttribute('data-client-id'))) > -1;
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
      var id = parseInt(el.getAttribute('data-client-id'));
      var isPinned = pins.indexOf(id) > -1;
      var cls = isPinned ? 'text-[var(--color-warning)]' : 'text-[var(--color-text-muted)]';
      star.outerHTML = heroicon("star", "text-sm", cls);
      if (isPinned) {
        btn.classList.add('bg-[var(--color-warning-light)]');
      } else {
        btn.classList.remove('bg-[var(--color-warning-light)]');
      }
    }
  });
  initSidebarVirtualScroll();
}
// === End pin & sort ===

// === Notification Context Menu & Delete ===
let notifCtxTarget = null, notifCtxId = null;

function deleteNotification(id, btn) {
  fetch('/business/notifications/' + id, {
    method: 'DELETE',
    headers: { 'X-CSRF-Token': getCookie('csrf_token') }
  }).then(function(r) { return r.json(); })
    .then(function(d) {
      if (d.success) {
        var row = btn.closest('[data-notif-id]') || btn;
        if (row) {
          row.style.opacity = 0;
          setTimeout(function() { row.remove(); }, 160);
        }
        showNotification('Notification deleted', 'success');
        var countEl = document.querySelector('.text-xs.text-[var(--color-text-muted)]');
        if (countEl && countEl.textContent.match(/^\d+ unread/)) {
          var n = parseInt(countEl.textContent); n = Math.max(n-1,0);
          countEl.textContent = n === 0 ? 'All read' : n + ' unread';
        }
      } else {
        showNotification(d.error || 'Failed to delete', 'error');
      }
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

// Chat menu dropdown (header menu button)
function waToggleChatMenu() {
  var dd = document.getElementById('chatMenuDropdown');
  if (dd) dd.classList.toggle('hidden');
}

function openAssistFromChat() {
  var dd = document.getElementById('chatMenuDropdown');
  if (dd) dd.classList.add('hidden');
  if (typeof toggleAssist === 'function') toggleAssist();
}

document.addEventListener('click', function(e) {
  var btn = document.getElementById('chatMenuBtn');
  var dd = document.getElementById('chatMenuDropdown');
  if (dd && !dd.classList.contains('hidden') && btn && !btn.contains(e.target) && !dd.contains(e.target)) {
    dd.classList.add('hidden');
  }
});
