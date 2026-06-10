;(function() {
  'use strict';

  var messageHistory = [];
  var MAX_HISTORY = 10;
  var ASSIST_KEY = 'salesmeeAssistHistory';
  var API_BASE = window.ASSIST_API_BASE || '/business/assist';

  function loadState() {
    try {
      var saved = localStorage.getItem(ASSIST_KEY);
      if (saved) {
        var parsed = JSON.parse(saved);
        if (Array.isArray(parsed)) messageHistory = parsed;
      }
    } catch(e) {}
  }

  function saveState() {
    try {
      localStorage.setItem(ASSIST_KEY, JSON.stringify(messageHistory));
    } catch(e) {}
  }

  function toggleAssist() {
    var panel = document.getElementById('assistPanel');
    var backdrop = document.getElementById('assistBackdrop');
    var btn = document.getElementById('assistButton');
    if (!panel) return;
    var isHidden = panel.classList.contains('hidden');
    panel.classList.toggle('hidden');
    if (backdrop) backdrop.classList.toggle('hidden');
    if (btn) btn.classList.toggle('active', isHidden);
    if (isHidden) {
      loadState();
      renderMessages();
      loadSuggestions();
      var input = document.getElementById('assistInput');
      if (input) setTimeout(function() { input.focus(); }, 300);
    }
  }

  function closeAssist() {
    var panel = document.getElementById('assistPanel');
    var backdrop = document.getElementById('assistBackdrop');
    var btn = document.getElementById('assistButton');
    if (panel && !panel.classList.contains('hidden')) {
      panel.classList.add('hidden');
      if (backdrop) backdrop.classList.add('hidden');
      if (btn) btn.classList.remove('active');
    }
  }

  function renderMessages() {
    var container = document.getElementById('assistMessages');
    if (!container) return;
    if (messageHistory.length === 0) {
      container.innerHTML = '<div class="assist-empty"><i class="fas fa-wand-magic-sparkles text-2xl mb-2" style="color:var(--color-primary)"></i><p class="text-sm font-medium" style="color:var(--color-text-secondary)">How can I help you?</p><p class="text-xs" style="color:var(--color-text-muted)">Ask me to draft replies, suggest products, or help with SalesMee.</p></div>';
      return;
    }
    var html = '';
    for (var i = 0; i < messageHistory.length; i++) {
      var msg = messageHistory[i];
      var isUser = msg.role === 'user';
      html += '<div class="assist-msg ' + (isUser ? 'assist-msg-user' : 'assist-msg-bot') + '">' +
        (isUser ? '' : '<div class="assist-avatar"><i class="fas fa-wand-magic-sparkles text-xs"></i></div>') +
        '<div class="assist-bubble">' + escapeHtml(msg.content) + '</div>' +
      '</div>';
    }
    container.innerHTML = html;
    container.scrollTop = container.scrollHeight;
  }

  function addMessage(role, content) {
    messageHistory.push({ role: role, content: content });
    if (messageHistory.length > MAX_HISTORY * 2) {
      messageHistory = messageHistory.slice(-MAX_HISTORY);
    }
    saveState();
    renderMessages();
  }

  function sendMessage(text) {
    if (!text || !text.trim()) return;
    var input = document.getElementById('assistInput');
    if (input) input.value = '';

    addMessage('user', text);

    var container = document.getElementById('assistMessages');
    var loadingEl = document.createElement('div');
    loadingEl.className = 'assist-msg assist-msg-bot';
    loadingEl.innerHTML = '<div class="assist-avatar"><i class="fas fa-wand-magic-sparkles text-xs"></i></div><div class="assist-bubble"><i class="fas fa-spinner fa-spin mr-2"></i>Thinking...</div>';
    container.appendChild(loadingEl);
    container.scrollTop = container.scrollHeight;

    var history = messageHistory.slice(-MAX_HISTORY);

    fetch(API_BASE + '/chat', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': getCookie('csrf_token') },
      body: JSON.stringify({ message: text, history: history })
    })
    .then(function(r) { return r.json(); })
    .then(function(data) {
      loadingEl.remove();
      if (data.reply) {
        addMessage('assistant', data.reply);
      } else {
        addMessage('assistant', 'Sorry, I had trouble processing that. Please try again.');
      }
    })
    .catch(function() {
      loadingEl.remove();
      addMessage('assistant', 'Sorry, I\'m temporarily unavailable. Please try again later.');
    });
  }

  function onAssistKeydown(e) {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      var input = document.getElementById('assistInput');
      if (input) sendMessage(input.value);
    }
  }

  function loadSuggestions() {
    fetch(API_BASE + '/suggestions')
      .then(function(r) { return r.json(); })
      .then(function(data) {
        var container = document.getElementById('assistSuggestions');
        if (!container || !data.suggestions) return;
        if (messageHistory.length > 0) {
          container.classList.add('hidden');
          return;
        }
        container.classList.remove('hidden');
        container.innerHTML = data.suggestions.map(function(s) {
          return '<button onclick="assistQuickAction(\'' + s.id + '\')" class="assist-chip" data-prompt="' + escapeHtml(s.prompt) + '">' +
            '<i class="fas fa-' + (s.id === 'draft-reply' ? 'feather' : s.id === 'suggest-product' ? 'wand-magic-sparkles' : s.id === 'help-platform' ? 'sparkles' : 'star') + ' mr-1.5 text-xs"></i>' +
            escapeHtml(s.label) + '</button>';
        }).join('');
      })
      .catch(function() {});
  }

  function assistQuickAction(id) {
    var container = document.getElementById('assistSuggestions');
    if (container) container.classList.add('hidden');
    var chip = document.querySelector('.assist-chip[onclick*="' + id + '"]');
    var prompt = chip ? chip.getAttribute('data-prompt') : '';
    var suffix = API_BASE === '/assist' ? '' : ' (regarding my business)';
    if (prompt) sendMessage(prompt + suffix);
  }

  function escapeHtml(text) {
    var div = document.createElement('div');
    div.appendChild(document.createTextNode(text));
    return div.innerHTML;
  }

  document.addEventListener('keydown', function(e) {
    if (e.key === 'Escape') {
      var panel = document.getElementById('assistPanel');
    if (panel) {
        closeAssist();
        e.stopPropagation();
      }
    }
  });

  loadState();

  window.toggleAssist = toggleAssist;
  window.closeAssist = closeAssist;
  window.sendAssistMessage = sendMessage;
  window.onAssistKeydown = onAssistKeydown;
  window.assistQuickAction = assistQuickAction;
})();
