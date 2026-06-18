;(function() {
  'use strict';

  var toastCounter = 0;

  window.escapeHtml = function escapeHtml(str) {
    if (str == null) return '';
    return String(str)
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#39;');
  };

  function playNotificationSound() {
    var enabled = localStorage.getItem('soundEnabled');
    if (enabled === 'false') return;
    try {
      var ctx = new (window.AudioContext || window.webkitAudioContext)();
      var g = ctx.createGain();
      g.connect(ctx.destination);
      g.gain.value = 0.12;

      var o1 = ctx.createOscillator();
      o1.type = 'sine';
      o1.frequency.value = 523.25;
      o1.connect(g);
      o1.start(ctx.currentTime);
      o1.stop(ctx.currentTime + 0.1);

      var o2 = ctx.createOscillator();
      o2.type = 'sine';
      o2.frequency.value = 659.25;
      o2.connect(g);
      o2.start(ctx.currentTime + 0.1);
      o2.stop(ctx.currentTime + 0.25);

      setTimeout(function() { ctx.close(); }, 500);
    } catch(e) {}
  }

  function showNotification(message, type) {
    type = type || 'info';
    var id = 'toast-' + (++toastCounter);
    var container = document.querySelector('.toast-container');
    if (!container) {
      container = document.createElement('div');
      container.className = 'toast-container';
      document.body.appendChild(container);
    }

    var icons = {
      success: 'fa-check-circle',
      error: 'fa-exclamation-circle',
      warning: 'fa-exclamation-triangle',
      info: 'fa-info-circle',
    };

    var colors = {
      success: 'text-emerald-500',
      error: 'text-rose-500',
      warning: 'text-amber-500',
      info: 'text-sky-500',
    };

    var titles = {
      success: 'Success',
      error: 'Error',
      warning: 'Warning',
      info: 'Notice',
    };

    var toast = document.createElement('div');
    toast.id = id;
    toast.className = 'toast toast-' + type;
    toast.innerHTML =
      '<i class="fas ' + (icons[type] || icons.info) + ' toast-icon ' + (colors[type] || colors.info) + '"></i>' +
      '<div class="toast-content">' +
        '<div class="toast-title">' + (titles[type] || titles.info) + '</div>' +
        '<div class="toast-message">' + escapeHtml(message) + '</div>' +
      '</div>' +
      '<button class="toast-close" onclick="removeToast(\'' + id + '\')" aria-label="Close">' +
        '<i class="fas fa-times text-xs"></i>' +
      '</button>' +
      '<div class="toast-progress" style="width:100%"></div>';

    playNotificationSound();
    container.appendChild(toast);
    requestAnimationFrame(function() {
      var progress = toast.querySelector('.toast-progress');
      if (progress) {
        progress.style.width = '0%';
        progress.style.transition = 'width 3s linear';
      }
    });

    setTimeout(function() {
      removeToast(id);
    }, 3200);
  }

  function removeToast(id) {
    var toast = document.getElementById(id);
    if (!toast) return;
    if (toast.classList.contains('toast-removing')) return;
    toast.classList.add('toast-removing');
    setTimeout(function() {
      if (toast.parentNode) toast.parentNode.removeChild(toast);
      var container = document.querySelector('.toast-container');
      if (container && container.children.length === 0) {
        document.body.removeChild(container);
      }
    }, 200);
  }

  function escapeHtml(text) {
    var div = document.createElement('div');
    div.appendChild(document.createTextNode(text));
    return div.innerHTML;
  }

  function getCookie(name) {
    var value = '; ' + document.cookie;
    var parts = value.split('; ' + name + '=');
    if (parts.length === 2) return parts.pop().split(';').shift();
  }

  function toggleHeaderMenu(btn) {
    var menu = btn.nextElementSibling;
    if (menu) {
      menu.classList.toggle('hidden');
    }
  }

  document.addEventListener('click', function(e) {
    if (e.target.closest && !e.target.closest('.relative.lg\\:hidden')) {
      var menus = document.querySelectorAll('#headerMenu:not(.hidden)');
      for (var i = 0; i < menus.length; i++) {
        menus[i].classList.add('hidden');
      }
    }
  });

  var modalCounter = 0;

  function showConfirmModal(options) {
    return new Promise(function(resolve) {
      var id = 'modal-' + (++modalCounter);
      var overlay = document.createElement('div');
      overlay.id = id;
      overlay.className = 'fixed inset-0 z-[100] flex items-center justify-center';
      overlay.style.background = 'rgba(0,0,0,0.5)';

      var title = options.title || 'Confirm';
      var message = options.message || 'Are you sure?';
      var confirmText = options.confirmText || 'Confirm';
      var cancelText = options.cancelText || 'Cancel';
      var confirmClass = options.confirmClass || 'bg-[var(--color-primary)] text-white';

      overlay.innerHTML =
        '<div class="bg-[var(--color-surface)] rounded-xl shadow-2xl p-5 mx-4 w-full max-w-sm border border-[var(--color-border)] animate-fade-in-down">' +
          '<h3 class="text-base font-semibold text-[var(--color-text)] mb-2">' + escapeHtml(title) + '</h3>' +
          '<p class="text-sm text-[var(--color-text-secondary)] mb-5">' + message + '</p>' +
          '<div class="flex gap-2 justify-end">' +
            '<button class="modal-cancel px-4 py-2 rounded-lg border border-[var(--color-border)] text-[var(--color-text)] hover:bg-[var(--color-surface-secondary)] text-sm font-medium transition">' + escapeHtml(cancelText) + '</button>' +
            '<button class="modal-confirm px-4 py-2 rounded-lg text-sm font-medium transition shadow-sm ' + confirmClass + ' hover:opacity-90">' + escapeHtml(confirmText) + '</button>' +
          '</div>' +
        '</div>';

      overlay.querySelector('.modal-cancel').addEventListener('click', function() {
        overlay.remove();
        resolve(false);
      });
      overlay.querySelector('.modal-confirm').addEventListener('click', function() {
        overlay.remove();
        resolve(true);
      });
      overlay.addEventListener('click', function(e) {
        if (e.target === overlay) {
          overlay.remove();
          resolve(false);
        }
      });

      document.body.appendChild(overlay);
    });
  }

  function showPromptModal(options) {
    return new Promise(function(resolve) {
      var id = 'modal-' + (++modalCounter);
      var overlay = document.createElement('div');
      overlay.id = id;
      overlay.className = 'fixed inset-0 z-[100] flex items-center justify-center';
      overlay.style.background = 'rgba(0,0,0,0.5)';

      var title = options.title || 'Input';
      var message = options.message || '';
      var defaultValue = options.defaultValue || '';
      var placeholder = options.placeholder || '';
      var confirmText = options.confirmText || 'OK';
      var cancelText = options.cancelText || 'Cancel';

      overlay.innerHTML =
        '<div class="bg-[var(--color-surface)] rounded-xl shadow-2xl p-5 mx-4 w-full max-w-sm border border-[var(--color-border)] animate-fade-in-down">' +
          '<h3 class="text-base font-semibold text-[var(--color-text)] mb-2">' + escapeHtml(title) + '</h3>' +
          (message ? '<p class="text-sm text-[var(--color-text-secondary)] mb-3">' + message + '</p>' : '') +
          '<input type="text" class="modal-input w-full px-3 py-2 rounded-lg border border-[var(--color-border)] bg-[var(--color-bg)] text-[var(--color-text)] text-sm focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)] mb-4" value="' + escapeHtml(defaultValue) + '" placeholder="' + escapeHtml(placeholder) + '">' +
          '<div class="flex gap-2 justify-end">' +
            '<button class="modal-cancel px-4 py-2 rounded-lg border border-[var(--color-border)] text-[var(--color-text)] hover:bg-[var(--color-surface-secondary)] text-sm font-medium transition">' + escapeHtml(cancelText) + '</button>' +
            '<button class="modal-confirm px-4 py-2 rounded-lg bg-[var(--color-primary)] text-white text-sm font-medium transition shadow-sm hover:opacity-90">' + escapeHtml(confirmText) + '</button>' +
          '</div>' +
        '</div>';

      var input = overlay.querySelector('.modal-input');
      input.focus();
      input.select();

      overlay.querySelector('.modal-cancel').addEventListener('click', function() {
        overlay.remove();
        resolve(null);
      });
      overlay.querySelector('.modal-confirm').addEventListener('click', function() {
        var val = input.value;
        overlay.remove();
        resolve(val);
      });
      input.addEventListener('keydown', function(e) {
        if (e.key === 'Enter') {
          var val = input.value;
          overlay.remove();
          resolve(val);
        }
        if (e.key === 'Escape') {
          overlay.remove();
          resolve(null);
        }
      });
      overlay.addEventListener('click', function(e) {
        if (e.target === overlay) {
          overlay.remove();
          resolve(null);
        }
      });

      document.body.appendChild(overlay);
    });
  }

  function showPaymentModal(maxAmount, currencySym, paymentMethods) {
    currencySym = currencySym || '$';
    return new Promise(function(resolve) {
      var id = 'modal-pay-' + (++modalCounter);
      var overlay = document.createElement('div');
      overlay.id = id;
      overlay.className = 'fixed inset-0 z-[100] flex items-center justify-center';
      overlay.style.background = 'rgba(0,0,0,0.5)';

      var hasMethods = paymentMethods && Array.isArray(paymentMethods) && paymentMethods.length > 0;

      function methodIcon(t) {
        switch(t) {
          case 'bank': return '🏦';
          case 'mobile_wallet': return '📱';
          case 'paypal': return '<img src=\"/static/images/paypal\" alt=\"PayPal\" class=\"w-4 h-4 object-contain\">';
          case 'link': return '🔗';
          case 'crypto': return '₿';
          case 'card': return '💳';
          case 'cash': return '💰';
          default: return '✏️';
        }
      }

      function methodDetail(t, d) {
        if (!d) return '';
        switch(t) {
          case 'bank': return d.bank_name || d.account_number || '';
          case 'mobile_wallet': return d.provider ? d.provider + (d.phone_number ? ' \u00b7 ' + d.phone_number : '') : (d.phone_number || d.upi_id || '');
          case 'paypal': return d.email || '';
          case 'link': return d.url || '';
          case 'crypto': return d.network || d.wallet_address || '';
          case 'card': return d.card_number || '';
          default: return '';
        }
      }

      function esc(t) {
        if (!t) return '';
        var e = document.createElement('div');
        e.textContent = String(t);
        return e.innerHTML;
      }

      var methodHtml = '';
      var selectedMethod = 'mobile_money';
      var amtMax = maxAmount.toFixed(2);

      if (hasMethods) {
        var cards = '';
        for (var i = 0; i < paymentMethods.length; i++) {
          var pm = paymentMethods[i];
          var icon = methodIcon(pm.method_type);
          var detail = methodDetail(pm.method_type, pm.details);
          if (i === 0) selectedMethod = pm.method_type;
          var selBorder = i === 0 ? 'var(--color-primary)' : 'var(--color-border)';
          var selBg = i === 0 ? 'var(--color-primary-light)' : 'transparent';
          cards += '<div class="method-card flex items-center gap-3 px-3 py-2.5 rounded-lg cursor-pointer transition-all border" data-method="' + pm.method_type + '" style="border-color:' + selBorder + ';background:' + selBg + '">' +
            '<div class="w-8 h-8 flex items-center justify-center rounded-lg bg-[var(--color-surface-secondary)] text-base flex-shrink-0">' + icon + '</div>' +
            '<div class="flex-1 min-w-0">' +
              '<p class="text-sm font-medium text-[var(--color-text)]">' + esc(pm.label) + '</p>' +
              (detail ? '<p class="text-xs text-[var(--color-text-secondary)] truncate">' + esc(detail) + '</p>' : '') +
            '</div>' +
            '<div class="method-radio w-4 h-4 rounded-full border-2 flex-shrink-0 transition-all" style="border-color:' + selBorder + ';background:' + (i === 0 ? selBorder : 'transparent') + '"></div>' +
          '</div>';
        }
        cards += '<div class="method-card flex items-center gap-3 px-3 py-2.5 rounded-lg cursor-pointer transition-all border" data-method="cash" style="border-color:var(--color-border);background:transparent">' +
          '<div class="w-8 h-8 flex items-center justify-center rounded-lg bg-[var(--color-surface-secondary)] text-base flex-shrink-0">💰</div>' +
          '<div class="flex-1 min-w-0">' +
            '<p class="text-sm font-medium text-[var(--color-text)]">Cash</p>' +
            '<p class="text-xs text-[var(--color-text-secondary)] truncate">Pay in person</p>' +
          '</div>' +
          '<div class="method-radio w-4 h-4 rounded-full border-2 flex-shrink-0 transition-all" style="border-color:var(--color-border);background:transparent"></div>' +
        '</div>';
        methodHtml = '<div class="method-cards space-y-1 mb-3">' + cards + '</div>';
        methodHtml += '<input type="hidden" class="modal-method" value="' + selectedMethod + '">';
      } else {
        methodHtml = '<select class="modal-method w-full px-3 py-2 rounded-lg border border-[var(--color-border)] bg-[var(--color-bg)] text-[var(--color-text)] text-sm focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)] mb-3">' +
          '<option value="mobile_money">Mobile Money</option>' +
          '<option value="bank_transfer">Bank Transfer</option>' +
          '<option value="card">Card Payment</option>' +
          '<option value="cash">Cash</option>' +
        '</select>';
      }

      overlay.innerHTML =
        '<div class="bg-[var(--color-surface)] rounded-xl shadow-2xl p-5 mx-4 w-full max-w-sm border border-[var(--color-border)]">' +
          '<div class="text-center mb-4">' +
            '<p class="text-xs font-medium text-[var(--color-text-secondary)] mb-1">Amount (' + currencySym + ')</p>' +
            '<input type="number" step="0.01" min="0.01" max="' + amtMax + '" class="modal-amount text-center text-3xl font-bold text-[var(--color-text)] bg-transparent border-none outline-none w-full" value="' + amtMax + '" placeholder="0.00">' +
            '<p class="text-xs text-[var(--color-text-secondary)] mt-0.5">Remaining: ' + currencySym + amtMax + '</p>' +
          '</div>' +
          '<label class="text-xs font-medium text-[var(--color-text-secondary)] block mb-2">Payment Method</label>' +
          methodHtml +
          '<label class="text-xs font-medium text-[var(--color-text-secondary)] block mb-1 mt-3">Reference (optional)</label>' +
          '<input type="text" class="modal-reference w-full px-3 py-2 rounded-lg border border-[var(--color-border)] bg-[var(--color-bg)] text-[var(--color-text)] text-sm focus:outline-none focus:ring-2 focus:ring-[var(--color-primary)] mb-4" placeholder="Transaction ID or note">' +
          '<div class="flex gap-2 justify-end">' +
            '<button class="modal-cancel px-4 py-2 rounded-lg border border-[var(--color-border)] text-[var(--color-text)] hover:bg-[var(--color-surface-secondary)] text-sm font-medium transition">Cancel</button>' +
            '<button class="modal-confirm px-4 py-2 rounded-lg bg-[var(--color-success)] text-white text-sm font-medium transition shadow-sm hover:opacity-90">Confirm Payment</button>' +
          '</div>' +
        '</div>';

      var amountInput = overlay.querySelector('.modal-amount');
      var refInput = overlay.querySelector('.modal-reference');
      var methodCards = overlay.querySelectorAll('.method-card');
      var hiddenInput = overlay.querySelector('.modal-method');

      if (methodCards.length > 0) {
        methodCards.forEach(function(c) {
          c.addEventListener('click', function() {
            methodCards.forEach(function(x) {
              x.style.borderColor = 'var(--color-border)';
              x.style.background = 'transparent';
              var r = x.querySelector('.method-radio');
              if (r) { r.style.borderColor = 'var(--color-border)'; r.style.background = 'transparent'; }
            });
            this.style.borderColor = 'var(--color-primary)';
            this.style.background = 'var(--color-primary-light)';
            var r = this.querySelector('.method-radio');
            if (r) { r.style.borderColor = 'var(--color-primary)'; r.style.background = 'var(--color-primary)'; }
            if (hiddenInput) hiddenInput.value = this.dataset.method;
          });
        });
      }

      amountInput.focus();
      amountInput.select();

      function getResult() {
        var amt = parseFloat(amountInput.value);
        if (isNaN(amt) || amt <= 0 || amt > maxAmount) {
          showNotification('Amount must be between ' + currencySym + '0.01 and ' + currencySym + maxAmount.toFixed(2), 'error');
          return null;
        }
        var method = hiddenInput ? hiddenInput.value : overlay.querySelector('.modal-method').value;
        return { amount: amt, method: method, reference: refInput.value };
      }

      overlay.querySelector('.modal-cancel').addEventListener('click', function() {
        overlay.remove();
        resolve(null);
      });
      overlay.querySelector('.modal-confirm').addEventListener('click', function() {
        var result = getResult();
        if (result) { overlay.remove(); resolve(result); }
      });
      amountInput.addEventListener('keydown', function(e) {
        if (e.key === 'Enter') {
          var result = getResult();
          if (result) { overlay.remove(); resolve(result); }
        }
        if (e.key === 'Escape') { overlay.remove(); resolve(null); }
      });
      overlay.addEventListener('click', function(e) {
        if (e.target === overlay) { overlay.remove(); resolve(null); }
      });

      document.body.appendChild(overlay);
    });
  }

  function showUpgradeModal(options) {
    return new Promise(function(resolve) {
      var id = 'upgrade-modal-' + (++modalCounter);
      var overlay = document.createElement('div');
      overlay.id = id;
      overlay.className = 'fixed inset-0 z-[100] flex items-center justify-center';
      overlay.style.background = 'rgba(0,0,0,0.5)';

      var title = options.title || 'Upgrade Required';
      var message = options.message || 'Your current plan does not support this action.';
      var upgradeUrl = options.upgradeUrl || '/business/subscription#plans';
      var graceAllowed = options.graceAllowed || false;
      var resource = options.resource || '';

      var upgradeBtn = '<a href="' + upgradeUrl + '" class="inline-flex items-center px-4 py-2 rounded-lg bg-[var(--color-primary)] text-white text-sm font-medium transition shadow-sm hover:opacity-90"><i class="fas fa-arrow-up mr-1.5 text-xs"></i> Upgrade Plan</a>';
      var graceBtn = graceAllowed ? '<button class="grace-continue px-4 py-2 rounded-lg border border-[var(--color-warning)] text-[var(--color-warning)] hover:bg-[var(--color-warning-light)] text-sm font-medium transition">Continue Once</button>' : '';
      var closeBtn = '<button class="modal-cancel px-4 py-2 rounded-lg border border-[var(--color-border)] text-[var(--color-text)] hover:bg-[var(--color-surface-secondary)] text-sm font-medium transition">Close</button>';

      overlay.innerHTML =
        '<div class="bg-[var(--color-surface)] rounded-xl shadow-2xl p-5 mx-4 w-full max-w-sm border border-[var(--color-border)] animate-fade-in-down">' +
          '<div class="flex items-center gap-3 mb-3">' +
            '<div class="w-10 h-10 rounded-full bg-[var(--color-warning-light)] flex items-center justify-center shrink-0">' +
              '<i class="fas fa-crown text-[var(--color-warning)]"></i>' +
            '</div>' +
            '<h3 class="text-base font-semibold text-[var(--color-text)]">' + escapeHtml(title) + '</h3>' +
          '</div>' +
          '<p class="text-sm text-[var(--color-text-secondary)] mb-5 leading-relaxed">' + message + '</p>' +
          '<div class="flex gap-2 justify-end flex-wrap">' +
            closeBtn +
            graceBtn +
            upgradeBtn +
          '</div>' +
        '</div>';

      overlay.querySelector('.modal-cancel').addEventListener('click', function() {
        overlay.remove();
        resolve(false);
      });
      var graceBtnEl = overlay.querySelector('.grace-continue');
      if (graceBtnEl) {
        graceBtnEl.addEventListener('click', function() {
          overlay.remove();
          resolve('grace');
        });
      }
      var upgradeBtnEl = overlay.querySelector('a[href]');
      if (upgradeBtnEl) {
        upgradeBtnEl.addEventListener('click', function() {
          overlay.remove();
          resolve('upgrade');
        });
      }
      overlay.addEventListener('click', function(e) {
        if (e.target === overlay) {
          overlay.remove();
          resolve(false);
        }
      });

      document.body.appendChild(overlay);
    });
  }

  window.handlePlanResponse = function(data, retryWithGrace) {
    if (data.limit_reached) {
      showUpgradeModal({
        title: 'Plan Limit Reached',
        message: data.error || 'You have reached the limit for your plan.',
        upgradeUrl: data.upgrade_url || '/business/subscription#plans',
        graceAllowed: data.grace_allowed,
      }).then(function(action) {
        if (action === 'grace' && retryWithGrace) {
          retryWithGrace();
        }
      });
      return true;
    }
    if (data.requires_upgrade) {
      showUpgradeModal({
        title: 'Upgrade Required',
        message: data.error || 'This feature requires an upgraded plan.',
        upgradeUrl: data.upgrade_url || '/business/subscription#plans',
        graceAllowed: false,
      });
      return true;
    }
    return false;
  };

  window.showUpgradeModal = showUpgradeModal;
  window.showConfirmModal = showConfirmModal;
  window.showPromptModal = showPromptModal;
  window.showPaymentModal = showPaymentModal;
  window.showNotification = showNotification;
  window.removeToast = removeToast;
  window.getCookie = getCookie;
  window.toggleHeaderMenu = toggleHeaderMenu;
  window.playNotificationSound = playNotificationSound;

  document.body.addEventListener('show-upgrade-modal', function(e) {
    showUpgradeModal({
      title: 'Upgrade Required',
      message: e.detail.message || 'Media sharing requires an upgraded plan.',
      upgradeUrl: e.detail.upgradeUrl || '/business/subscription#plans',
      graceAllowed: false,
    });
  });
})();
