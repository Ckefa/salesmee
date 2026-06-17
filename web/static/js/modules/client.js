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

function loadBusiness(businessId) {
  currentBusinessId = businessId;
  window.businessId = businessId;
  document.querySelectorAll('.business-item').forEach(item => {
    item.classList.remove('selected');
  });
  const el = document.querySelector(`[data-business-id="${businessId}"]`);
  if (el) el.classList.add('selected');
  htmx.ajax('GET', `/client/businesses/${businessId}/messages`, {
    target: '#chat-area',
    swap: 'innerHTML'
  });
  // On mobile, show chat area (replaces sidebar entirely)
  var layout = document.getElementById('clientLayout');
  if (layout) {
    layout.classList.add('wa-chat-open');
  }
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
