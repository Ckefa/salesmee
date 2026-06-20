let currentBusinessId = null;
let bizCtxBusinessId = null;
let bizCtxBusinessName = '';

document.addEventListener('DOMContentLoaded', function () {
  document.addEventListener('click', function (e) {
    hideBizCtxMenu();
    const item = e.target.closest('.business-item');
    if (item && !e.target.closest('.pin-btn') && !e.target.closest('.wa-chat-icon-btn')) {
      loadBusiness(item.getAttribute('data-business-id'));
    }
  });
  document.addEventListener('contextmenu', function (e) {
    if (!e.target.closest('.business-item') && !e.target.closest('#bizCtxMenu')) {
      hideBizCtxMenu();
    }
  });

  // Long-press on business list items (touch)
  var bizList = document.getElementById('business-list');
  if (bizList) {
    var longTimer = null;
    bizList.addEventListener('touchstart', function(e) {
      var item = e.target.closest('.business-item');
      if (item && e.touches.length === 1) {
        var id = item.getAttribute('data-business-id');
        var name = item.getAttribute('data-business-name') || '';
        if (id) {
          longTimer = setTimeout(function() {
            showBizCtxMenu({clientX: e.touches[0].clientX, clientY: e.touches[0].clientY, preventDefault: function(){}}, id, name);
          }, 500);
        }
      }
    });
    bizList.addEventListener('touchend', function() { clearTimeout(longTimer); });
    bizList.addEventListener('touchmove', function() { clearTimeout(longTimer); });
  }

  startPresenceWS();
});

function registerClientPresenceHandlers() {
  if (window._clientPresenceRegistered) return;
  window._clientPresenceRegistered = true;
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
}

function startPresenceWS() {
  if (window.wsClient) {
    registerClientPresenceHandlers();
    return;
  }
  var token = getCookie('client_token');
  if (!token) return;
  window.wsClient = new WsClient();
  window.wsClient.connect('/ws/client?token=' + encodeURIComponent(token));
  registerClientPresenceHandlers();
}

window.addEventListener('beforeunload', function() {
  if (window.wsClient) {
    window.wsClient.disconnect();
    window.wsClient = null;
  }
});

function showBizCtxMenu(e, bizId, bizName) {
  e.preventDefault();
  bizCtxBusinessId = bizId;
  bizCtxBusinessName = bizName;
  var menu = document.getElementById('bizCtxMenu');
  if (!menu) return;
  // Update pin label based on current state
  var pins = JSON.parse(localStorage.getItem('pinned_businesses') || '[]');
  var pinBtn = menu.querySelector('[data-action="toggle-pin"]');
  if (pinBtn) {
    var isPinned = pins.indexOf(bizId) > -1;
    pinBtn.innerHTML = (isPinned ? '<i class="fas fa-star"></i><span>Unpin</span>' : '<i class="fas fa-star"></i><span>Pin to top</span>');
  }
  var x = e.clientX, y = e.clientY;
  var w = window.innerWidth, h = window.innerHeight;
  var mw = 200, mh = menu.offsetHeight || 128;
  if (x + mw > w) x = w - mw - 8;
  if (y + mh > h) y = h - mh - 8;
  if (x < 8) x = 8;
  if (y < 8) y = 8;
  menu.style.left = x + 'px';
  menu.style.top = y + 'px';
  menu.classList.remove('hidden');
}

function hideBizCtxMenu() {
  var menu = document.getElementById('bizCtxMenu');
  if (menu) menu.classList.add('hidden');
  bizCtxBusinessId = null;
  bizCtxBusinessName = '';
}

function bizCtxMarkRead() {
  if (!bizCtxBusinessId) return;
  fetch('/client/businesses/' + bizCtxBusinessId + '/read', {
    method: 'PUT',
    headers: { 'X-CSRF-Token': getCookie('csrf_token') }
  }).then(function() {
    var badge = document.querySelector('.business-item[data-business-id="' + bizCtxBusinessId + '"] .wa-unread-badge');
    if (badge) badge.remove();
    showNotification('Marked as read', 'success');
  }).catch(console.error);
  hideBizCtxMenu();
}

function bizCtxTogglePin() {
  if (!bizCtxBusinessId) return;
  togglePinBusiness(bizCtxBusinessId);
  hideBizCtxMenu();
}

function bizCtxRemove() {
  if (!bizCtxBusinessId) return;
  disconnectBusiness(bizCtxBusinessId);
  hideBizCtxMenu();
}

function buildClientSkeletonChatContainer(businessId) {
  var el = document.querySelector('[data-business-id="' + businessId + '"]');
  var name = el ? el.getAttribute('data-business-name') || 'Loading...' : 'Loading...';
  var type = el ? el.getAttribute('data-business-type') || '' : '';
  return '<div class="wa-chat-container">' +
    '<div class="wa-chat-header">' +
      '<button onclick="clientBackFromChat()" class="wa-chat-back" title="Back"><i class="fas fa-arrow-left"></i></button>' +
      '<div class="wa-chat-header-avatar">' +
        '<div class="wa-chat-avatar wa-sidebar-avatar-placeholder"><i class="fas fa-store"></i></div>' +
      '</div>' +
      '<div class="wa-chat-header-info">' +
        '<div class="wa-chat-header-name">' + escapeHtml(name) + '</div>' +
        '<div class="wa-chat-header-status flex items-center gap-1.5">' +
          '<span class="w-1.5 h-1.5 rounded-full skeleton" style="display:inline-block"></span>' +
          '<span class="text-xs text-[var(--color-text-muted)]">' + escapeHtml(type) + '</span>' +
        '</div>' +
      '</div>' +
      '<div class="flex items-center gap-1 ml-auto">' +
        '<div class="skeleton rounded-lg h-8 w-12"></div>' +
        '<div class="skeleton rounded-lg h-8 w-12"></div>' +
        '<div class="skeleton rounded-lg h-8 w-12"></div>' +
      '</div>' +
    '</div>' +
    '<div id="chat-content" style="flex:1;min-height:0;display:flex;flex-direction:column">' +
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
    '</div>' +
  '</div>';
}

function loadBusiness(businessId) {
  currentBusinessId = businessId;
  window.businessId = businessId;
  document.querySelectorAll('.business-item').forEach(function(item) {
    item.classList.remove('selected');
  });
  var el = document.querySelector('[data-business-id="' + businessId + '"]');
  if (el) el.classList.add('selected');
  window.conversationId = el ? el.getAttribute('data-conversation-id') : '';

  // Build chat container with real header + skeleton messages
  var contentArea = document.getElementById('content-area');
  contentArea.innerHTML = buildClientSkeletonChatContainer(businessId);
  contentArea.classList.add('content-fade-in');

  // On mobile, show chat area
  var layout = document.getElementById('clientLayout');
  if (layout) {
    layout.classList.add('wa-chat-open');
  }

  var loadTimeout = setTimeout(function() {
    var mc = document.getElementById('messages-container');
    if (mc && mc.querySelector('.skeleton')) {
      mc.innerHTML = '<div class="flex flex-col items-center justify-center py-12 text-center flex-1">' +
        '<div class="w-14 h-14 rounded-full bg-[var(--color-error-light)] flex items-center justify-center mb-4">' +
        '<i class="fas fa-exclamation-triangle text-[var(--color-error)] text-2xl"></i></div>' +
        '<p class="text-sm font-medium text-[var(--color-text)] mb-1">Failed to load</p>' +
        '<p class="text-xs text-[var(--color-text-muted)] mb-4">Timed out. Please try again.</p>' +
        '<button onclick="loadBusiness(' + businessId + ')" class="px-4 py-2 bg-[var(--color-primary)] text-white rounded-lg text-sm hover:opacity-90 transition-colors">' +
        '<i class="fas fa-refresh mr-1"></i> Retry</button></div>';
    }
  }, 20000);

  fetch('/client/businesses/' + businessId + '/messages', {
    headers: { 'X-CSRF-Token': getCookie('csrf_token') }
  })
    .then(function(r) { return r.text(); })
    .then(function(html) {
      clearTimeout(loadTimeout);
      var parser = new DOMParser();
      var doc = parser.parseFromString(html, 'text/html');
      // Swap messages-container
      var newMessages = doc.getElementById('messages-container');
      if (newMessages) {
        var oldMessages = document.getElementById('messages-container');
        if (oldMessages) oldMessages.outerHTML = newMessages.outerHTML;
      }
      // Swap scroll button
      var newScrollBtn = doc.getElementById('scrollToBottom');
      if (newScrollBtn) {
        var oldBtn = document.getElementById('scrollToBottom');
        if (oldBtn) oldBtn.outerHTML = newScrollBtn.outerHTML;
      }
      // Swap input
      var newInput = doc.querySelector('.wa-input-wrapper');
      if (newInput) {
        var oldInput = document.querySelector('.wa-input-wrapper');
        if (oldInput) oldInput.outerHTML = newInput.outerHTML;
      }
      // Swap header
      var newHeader = doc.querySelector('.wa-chat-header');
      if (newHeader) {
        var oldHeader = document.querySelector('.wa-chat-header');
        if (oldHeader) oldHeader.outerHTML = newHeader.outerHTML;
      }
      htmx.process(contentArea);
      scrollToBottom();
      if (typeof markAsRead === 'function') markAsRead();
      initOlderObserver();
      initScrollToBottom();
    })
    .catch(function() {
      clearTimeout(loadTimeout);
      var mc = document.getElementById('messages-container');
      if (mc) {
        mc.innerHTML = '<div class="flex flex-col items-center justify-center py-12 text-center flex-1">' +
          '<div class="w-14 h-14 rounded-full bg-[var(--color-error-light)] flex items-center justify-center mb-4">' +
          '<i class="fas fa-exclamation-triangle text-[var(--color-error)] text-2xl"></i></div>' +
          '<p class="text-sm font-medium text-[var(--color-text)] mb-1">Failed to load</p>' +
          '<p class="text-xs text-[var(--color-text-muted)] mb-4">Something went wrong.</p>' +
          '<button onclick="loadBusiness(' + businessId + ')" class="px-4 py-2 bg-[var(--color-primary)] text-white rounded-lg text-sm hover:opacity-90 transition-colors">' +
          '<i class="fas fa-refresh mr-1"></i> Retry</button></div>';
      }
      showNotification('Failed to load conversation', 'error');
    });
}

function disconnectBusiness(businessId) {
  showConfirmModal({
    title: 'Remove Business',
    message: 'Remove this business from your list? You can reconnect later.',
    confirmText: 'Remove',
    confirmClass: 'bg-[var(--color-error)] text-white'
  }).then(function(confirmed) {
    if (!confirmed) return;
    fetch('/client/disconnect/' + businessId, { method: 'POST', headers: { 'X-CSRF-Token': getCookie('csrf_token') } })
      .then(r => r.json())
      .then(data => {
        if (data.success) {
          htmx.ajax('GET', window.location.href, {target: 'body', swap: 'innerHTML'});
        } else {
          showNotification('Failed to remove business', 'error');
        }
      })
      .catch(() => showNotification('Failed to remove business', 'error'));
  });
}

function hideClientOrderModal() {
  document.getElementById('clientOrderModal')?.classList.add('hidden');
  document.getElementById('clientOrderForm')?.reset();
}

function submitOrderForm() {
  const productSelect = document.getElementById('clientOrderProduct');
  const quantityInput = document.getElementById('clientOrderQuantity');
  if (!productSelect.value) return showNotification('Please select a product', 'error');
  if (!quantityInput.value || quantityInput.value < 1) return showNotification('Please enter a valid quantity', 'error');

  const data = {
    product_id: parseInt(productSelect.value),
    quantity: parseInt(quantityInput.value),
    delivery_address: document.getElementById('clientOrderAddress').value,
    notes: document.getElementById('clientOrderNotes').value,
    business_id: parseInt(currentBusinessId)
  };

  fetch('/client/orders', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': getCookie('csrf_token') },
    body: JSON.stringify(data)
  })
    .then(r => r.json())
    .then(data => {
      hideClientOrderModal();
      showNotification('Order request sent successfully! Redirecting to chat...', 'success');
      setTimeout(() => window.location.href = `/client/businesses/${currentBusinessId}/messages`, 1500);
    })
    .catch(e => { console.error(e); showNotification('Failed to send order request', 'error'); });
}
