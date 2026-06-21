// chat_common.js — Shared chat functions for business and client

var MAX_DOM_MESSAGES = 200;
var detachedMessages = {};

if (window.typingTimeout) clearTimeout(window.typingTimeout);
var typingTimeout = null;

function validateHtmxRequest(evt) {
  var form = evt.detail.elt;
  var input = form.querySelector('#messageInput');
  var message = input ? input.value.trim() : '';
  var hasFile = false;
  var fileInputs = form.querySelectorAll('input[type="file"]');
  for (var i = 0; i < fileInputs.length; i++) {
    if (fileInputs[i].files && fileInputs[i].files.length > 0) {
      hasFile = true;
      break;
    }
  }
  if (!message && !hasFile) {
    if (typeof showNotification === 'function') showNotification('Please enter a valid message', 'warning');
    evt.preventDefault();
  }
}

// ========== Optimistic Message Send ==========

function renderOwnMessage(content, tempId, mediaInfo) {
  var timeStr = new Date().toLocaleTimeString([], { hour: 'numeric', minute: '2-digit' });
  var mediaHtml = '';
  if (mediaInfo && mediaInfo.url) {
    if (mediaInfo.type === 'image') {
      mediaHtml = '<img src="' + mediaInfo.url + '" alt="Image" class="wa-media-image" style="max-width:200px;max-height:200px;object-fit:cover;border-radius:8px;opacity:0.7;">';
    } else {
      mediaHtml = '<div class="wa-media-doc">' + heroicon('document-text', 'wa-media-doc-icon') + '<span class="wa-media-doc-link">' + escapeHtml(mediaInfo.name) + '</span><span class="text-xs text-[var(--color-text-muted)] ml-2">uploading...</span></div>';
    }
  }
  var bodyHtml = mediaHtml + (content ? '<span class="msg-txt">' + escapeHtml(content) + '</span>' : '');
  return '<div class="msg out message-item optimistic" data-message-id="temp-' + tempId + '">'
    + '<div class="msg-bbl">'
    + '<svg class="msg-tail" viewBox="0 0 10 15" height="15" width="10" preserveAspectRatio="xMidYMid meet">'
    + '<path fill="var(--color-bg)" d="M9,3L0,14V1H7C8.5,1,9.5,2,9,3z"></path>'
    + '<path fill="currentColor" d="M9,2L0,13V0H7C8.5,0,9.5,1,9,2z"></path></svg>'
    + bodyHtml
    + '<span class="msg-meta"><span class="msg-time">' + timeStr + '</span>'
    + '<span class="msg-tick inline-flex items-center ml-0.5 align-middle" data-read-state="sent" style="width:12px;height:11px;">'
    + '<svg viewBox="0 0 12 12" width="12" height="11" fill="none" stroke="#8696a0" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">'
    + '<path d="M2 6L5 9L11 3"/></svg></span></span></div></div>';
}

function optimisticSend(evt) {
  var form = evt.detail.elt;
  if (!form || form.id !== 'message-form') return;
  var input = form.querySelector('#messageInput');
  var content = input ? input.value.trim() : '';
  var mediaInfo = null;
  var fileInputs = form.querySelectorAll('input[type="file"]');
  for (var i = 0; i < fileInputs.length; i++) {
    if (fileInputs[i].files && fileInputs[i].files.length > 0) {
      var file = fileInputs[i].files[0];
      var type = fileInputs[i].name.replace('media_', '');
      var url = type === 'image' ? URL.createObjectURL(file) : null;
      mediaInfo = { type: type, name: file.name, url: url };
      break;
    }
  }
  window._optTempId = (window._optTempId || 0) + 1;
  var tempId = window._optTempId;
  var container = document.getElementById('messages-container');
  if (!container) return;
  container.insertAdjacentHTML('beforeend', renderOwnMessage(content, tempId, mediaInfo));
  requestAnimationFrame(function() { container.scrollTop = container.scrollHeight; });
}

function confirmSend(evt) {
  var xhr = evt.detail.xhr;
  var tempId = window._optTempId;
  if (!tempId) return;
  var tempEl = document.querySelector('[data-message-id="temp-' + tempId + '"]');
  if (!tempEl) return;
  if (xhr.status >= 200 && xhr.status < 300) {
    var html = xhr.responseText;
    if (html && html.trim().length > 0) {
      tempEl.outerHTML = html;
    } else {
      tempEl.remove();
    }
  } else {
    tempEl.classList.add('opacity-50');
    var txt = tempEl.querySelector('.msg-txt');
    if (txt) txt.innerHTML += ' <span class="text-[var(--color-error)] text-xs">Failed</span>';
  }
}

function validateAndOptimisticSend(evt) {
  validateHtmxRequest(evt);
  if (!evt.defaultPrevented) {
    optimisticSend(evt);
  }
}

var unreadBelow = 0;
var MAX_DOM_MESSAGES = 200;
var detachedMessages = {};
window.clearUnreadBelow = function() {
  if (unreadBelow > 0 && typeof markAsRead === 'function') markAsRead();
  unreadBelow = 0;
  var badge = document.getElementById('scrollBottomBadge');
  if (badge) badge.classList.remove('visible');
};

function updateScrollBottomBadge() {
  var badge = document.getElementById('scrollBottomBadge');
  if (!badge) return;
  if (unreadBelow > 0) {
    badge.textContent = unreadBelow > 99 ? '99+' : unreadBelow;
    badge.classList.add('visible');
  } else {
    badge.classList.remove('visible');
  }
}

function scrollToBottom() {
  var container = document.getElementById('messages-container');
  if (container) requestAnimationFrame(function() {
    container.scrollTop = container.scrollHeight;
  });
}

function markAsRead() {
  var id = window.clientId || window.currentClientId;
  if (!id) return;
  fetch(`/business/clients/${id}/read`, { method: 'PUT', headers: { 'X-CSRF-Token': getCookie('csrf_token') } })
    .then(function() {
      var badge = document.querySelector('.client-item[data-client-id="' + id + '"] .wa-unread-badge');
      if (badge) badge.remove();
    })
    .catch(console.error);
}

function tickSvg(state) {
  if (state === 'read') {
    return '<svg viewBox="0 0 16 12" width="14" height="11" fill="none" stroke="#53bdeb" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M2 6L5 9L11 3"/><path d="M6 6L9 9L15 3"/></svg>';
  }
  if (state === 'delivered') {
    return '<svg viewBox="0 0 16 12" width="14" height="11" fill="none" stroke="#8696a0" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M2 6L5 9L11 3"/><path d="M6 6L9 9L15 3"/></svg>';
  }
  return '<svg viewBox="0 0 12 12" width="12" height="11" fill="none" stroke="#8696a0" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M2 6L5 9L11 3"/></svg>';
}

function setMessageTickState(item, state) {
  if (!item) return;
  var tick = item.querySelector('.msg-tick');
  if (!tick) return;
  if (tick.getAttribute('data-read-state') === 'read') return;
  tick.setAttribute('data-read-state', state);
  tick.innerHTML = tickSvg(state);
  tick.style.width = state === 'sent' ? '12px' : '14px';
}

function applyReadReceipt(receipt) {
  if (!receipt || receipt.reader_type === 'business') return;
  var cid = window.conversationId;
  if (!cid) return;
  if (receipt.conversation_id && receipt.conversation_id !== String(cid)) return;

  if (receipt.message_id) {
    setMessageTickState(document.querySelector('#messages-container .message-item.out[data-message-id="' + receipt.message_id + '"]'), 'read');
    return;
  }

  document.querySelectorAll('#messages-container .message-item.out').forEach(function(item) {
    setMessageTickState(item, 'read');
  });
}

function markVisibleConversationRead() {
  if (document.visibilityState === 'hidden') return;
  markAsRead();
}

function starsHtml(rating) {
  var html = '';
  var r = rating || 5;
  for (var i = 1; i <= 5; i++) {
    html += i <= r ? heroicon('star') : heroicon('star');
  }
  return html;
}

function addReviewBadgeToCard(card, rating) {
  if (!card || card.querySelector('[data-review-badge]')) return;
  var badge = document.createElement('div');
  badge.setAttribute('data-review-badge', '1');
  badge.className = 'w-full mt-2 py-1.5 px-3 rounded-lg bg-[var(--color-warning-light)] text-[var(--color-warning)] text-xs font-medium text-center flex items-center justify-center gap-0.5';
  badge.innerHTML = starsHtml(rating) + '<span class="ml-1.5 text-[var(--color-text-secondary)]">Reviewed</span>';
  var timestamp = card.querySelector('.mt-2.text-right');
  if (timestamp) card.insertBefore(badge, timestamp);
  else card.appendChild(badge);
}

function renderMediaMessage(msg) {
  var url = '/static/' + escapeHtml(msg.media_url);
  var mediaTag = '';
  if (msg.media_type === 'image') {
    mediaTag = '<img src="' + url + '" alt="Image" class="wa-media-image" onclick="window.open(this.src)" loading="lazy">';
  } else if (msg.media_type === 'document') {
    mediaTag = '<div class="wa-media-doc">' + heroicon('document-text', 'wa-media-doc-icon') + '<a href="' + url + '" target="_blank" class="wa-media-doc-link">' + escapeHtml(msg.media_url.split('/').pop()) + '</a></div>';
  } else if (msg.media_type === 'audio') {
    mediaTag = '<div class="wa-media-audio"><audio controls class="wa-audio-player" preload="metadata"><source src="' + url + '"></audio></div>';
  } else {
    mediaTag = '<a href="' + url + '" target="_blank" class="wa-media-doc-link">' + heroicon('document') + ' ' + escapeHtml(msg.media_url.split('/').pop()) + '</a>';
  }
  var inner = mediaTag + (msg.content ? '<p>' + escapeHtml(msg.content) + '</p>' : '') + '<span class="msg-meta"><span class="msg-time">' + formatTime(msg.created_at) + '</span></span>';
  return '<div class="msg in message-item" data-message-id="' + msg.id + '"><div class="msg-bbl" style="padding:3px;"><svg class="msg-tail" viewBox="0 0 10 15" height="15" width="10" preserveAspectRatio="xMidYMid meet"><path fill="var(--color-bg)" d="M1,3L10,14V1H3C1.5,1,0.5,2,1,3z"></path><path fill="currentColor" d="M1,2L10,13V0H3C1.5,0,0.5,1,1,2z"></path></svg>' + inner + '</div></div>';
}

function escapeHtml(str) {
  if (!str) return '';
  var div = document.createElement('div');
  div.appendChild(document.createTextNode(str));
  return div.innerHTML;
}

function formatTime(ts) {
  if (!ts) return '';
  var d = new Date(Number(ts));
  return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
}

function myUserType() {
  return (typeof sender !== 'undefined' && sender === 'business') ? 'business' : 'client';
}

function showTypingIndicator(typing) {
  if (!typing || !window.conversationId) return;
  if (typing.conversation_id !== String(window.conversationId)) return;
  if (typing.user_type === myUserType()) return;
  var el = document.getElementById('typingIndicator');
  if (!el) {
    el = document.createElement('div');
    el.id = 'typingIndicator';
    el.className = 'msg in typing-indicator';
    el.innerHTML = '<div class="msg-bbl typing"><span class="typing-label">typing</span><span class="typing-dot"></span><span class="typing-dot"></span><span class="typing-dot"></span></div>';
    document.getElementById('messages-container').appendChild(el);
  }
}

function hideTypingIndicator(typing) {
  if (!typing || !window.conversationId) return;
  if (typing.conversation_id !== String(window.conversationId)) return;
  if (typing.user_type === myUserType()) return;
  var el = document.getElementById('typingIndicator');
  if (el) el.remove();
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
        var el = document.querySelector('[data-message-id="' + id + '"]');
        if (el) el.remove();
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
  if (window.wsClient) window.wsClient.disconnect();
});

document.addEventListener('visibilitychange', function() {
  if (document.visibilityState === 'visible') {
    var container = document.getElementById('messages-container');
    if (!container) return;
    var isNearBottom = container.scrollHeight - container.scrollTop - container.clientHeight < 150;
    if (isNearBottom && typeof markAsRead === 'function') markAsRead();
  }
});

// ========== Older Message Pagination (IntersectionObserver) ==========

var olderObserver = null;

function initOlderObserver() {
  if (olderObserver) {
    olderObserver.disconnect();
    olderObserver = null;
  }
  var sentinel = document.getElementById('load-older-sentinel');
  if (!sentinel) return;
  olderObserver = new IntersectionObserver(function(entries) {
    if (entries[0].isIntersecting) {
      loadOlderMessages();
    }
  }, { rootMargin: '200px 0px 0px 0px' });
  olderObserver.observe(sentinel);
}

function buildMessagesUrl() {
  var container = document.getElementById('messages-container');
  if (!container) return '';
  var cursor = container.getAttribute('data-older-cursor');
  if (!cursor) return '';
  var isBusiness = typeof window.clientId !== 'undefined' && window.clientId;
  if (isBusiness) {
    return 'clients/' + window.clientId + '/messages?before=' + encodeURIComponent(cursor) + '&limit=50';
  }
  if (window.businessId) {
    return '/client/businesses/' + window.businessId + '/messages?before=' + encodeURIComponent(cursor) + '&limit=50';
  }
  return '';
}

function recycleMessages(container) {
  var cid = window.conversationId;
  if (!cid) return;
  var messages = container.querySelectorAll('[data-message-id]');
  if (messages.length <= MAX_DOM_MESSAGES) return;

  var excess = messages.length - MAX_DOM_MESSAGES;
  var frag = document.createDocumentFragment();
  for (var i = 0; i < excess; i++) {
    var el = messages[i];
    if (el.id === 'load-older-sentinel') { excess++; continue; }
    frag.appendChild(el);
  }
  if (frag.children.length > 0) {
    detachedMessages[cid] = frag;
  }
}

function loadOlderMessages() {
  if (window._loadingOlder) return;
  var container = document.getElementById('messages-container');
  if (!container) return;
  if (container.getAttribute('data-has-more') !== 'true') return;

  if (olderObserver) {
    olderObserver.disconnect();
    olderObserver = null;
  }

  window._loadingOlder = true;
  var url = buildMessagesUrl();
  if (!url) { window._loadingOlder = false; return; }

  fetch(url)
    .then(function(r) { return r.text(); })
    .then(function(html) {
      var parser = new DOMParser();
      var doc = parser.parseFromString(html, 'text/html');
      var nextContainer = doc.getElementById('messages-container');
      if (!nextContainer) { window._loadingOlder = false; return; }

      container.setAttribute('data-has-more', nextContainer.getAttribute('data-has-more') || 'false');
      container.setAttribute('data-older-cursor', nextContainer.getAttribute('data-older-cursor') || '');

      // Extract message items from response
      var items = nextContainer.querySelectorAll('[data-message-id]');
      var firstChild = container.firstChild;

      items.forEach(function(el) {
        container.insertBefore(el.cloneNode(true), firstChild);
      });

      if (typeof htmx !== 'undefined') {
        htmx.process(container);
      }

      recycleMessages(container);

      window._loadingOlder = false;
      initOlderObserver();
    })
    .catch(function() {
      window._loadingOlder = false;
      initOlderObserver();
    });
}

document.addEventListener('DOMContentLoaded', function() {
  initOlderObserver();
});

// Re-init observer after HTMX swaps the chat area (client-side navigation)
document.addEventListener('htmx:afterSwap', function(e) {
  if (e.detail.target && e.detail.target.id === 'chat-area') {
    initOlderObserver();
  }
});

// ========== Scroll-to-Bottom Button ==========

window.scrollToBottomBtn = function() {
  var c = document.getElementById('messages-container');
  if (c) c.scrollTop = c.scrollHeight;
  if (window.clearUnreadBelow) window.clearUnreadBelow();
  if (typeof markAsRead === 'function') markAsRead();
};

function initScrollToBottom() {
  var container = document.getElementById('messages-container');
  var btn = document.getElementById('scrollToBottom');
  if (!container || !btn) return;
  if (container.getAttribute('data-scroll-init')) return;
  container.setAttribute('data-scroll-init', 'true');
  var toggle = function() {
    var isNearBottom = container.scrollHeight - container.scrollTop - container.clientHeight < 150;
    btn.classList.toggle('visible', !isNearBottom);
    if (isNearBottom && window.clearUnreadBelow) window.clearUnreadBelow();
  };
  container.addEventListener('scroll', toggle);
  setTimeout(toggle, 100);
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

  // Typing indicator via WebSocket
  if (window.wsClient && window.wsClient.isConnected) {
    if (!window.conversationId) return;
    if (typingTimeout) clearTimeout(typingTimeout);
    var myType = myUserType();
    var myId = myType === 'business' ? businessId : clientId;
    if (val.length > 0) {
      window.wsClient.sendTypingStart(conversationId, myId, myType, clientId, businessId);
    } else {
      window.wsClient.sendTypingStop(conversationId, myId, myType, clientId, businessId);
    }
    typingTimeout = setTimeout(function() {
      window.wsClient.sendTypingStop(conversationId, myId, myType, clientId, businessId);
    }, 3000);
  }
}

function onMessageKeydown(event) {
  var qr = document.getElementById('quickReplies');
  if (event.key === 'Escape' && qr && !qr.classList.contains('hidden')) {
    qr.classList.add('hidden');
    var input = document.getElementById('messageInput');
    if (input) input.value = input.value.replace(/\/$/, '');
  }

  // Send typing stop on Enter (message sent)
  if (event.key === 'Enter' && !event.shiftKey) {
    if (window.wsClient && window.wsClient.isConnected) {
      if (!window.conversationId) return;
      var myType = myUserType();
      var myId = myType === 'business' ? businessId : clientId;
      window.wsClient.sendTypingStop(conversationId, myId, myType, clientId, businessId);
    }
    if (typingTimeout) {
      clearTimeout(typingTimeout);
      typingTimeout = null;
    }
  }
}
