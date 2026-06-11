;(function() {
  'use strict';

  var pollInterval = null;
  var isChecking = false;

  function showStepCompleteAndAdvance() {
    var container = document.getElementById('onboardingStepContainer');
    var steps = container.querySelectorAll('.onboarding-step');
    steps.forEach(function(s) { s.classList.add('hidden'); });
    document.getElementById('onboardingStepComplete').classList.remove('hidden');
    setTimeout(function() {
      window.location.reload();
    }, 1200);
  }

  function csrfFetch(url, options) {
    options = options || {};
    options.headers = options.headers || {};
    options.headers['X-CSRF-Token'] = getCookie('csrf_token');
    return fetch(url, options);
  }

  window.onboardingAdvance = function() {
    csrfFetch('/business/onboarding/advance', { method: 'POST' })
    .then(function(r) { return r.json(); })
    .then(function(data) {
      if (data.completed) {
        showStepCompleteAndAdvance();
      } else if (data.step) {
        showStepCompleteAndAdvance();
      }
    })
    .catch(function() {});
  };

  window.onboardingCheck = function() {
    if (isChecking) return;
    isChecking = true;
    csrfFetch('/business/onboarding/progress', { method: 'POST' })
    .then(function(r) { return r.json(); })
    .then(function(data) {
      isChecking = false;
      if (data.completed) {
        showStepCompleteAndAdvance();
      } else if (data.advanced) {
        showStepCompleteAndAdvance();
      } else if (data.message) {
        if (typeof showNotification !== 'undefined') {
          showNotification(data.message, 'info');
        }
      }
    })
    .catch(function() { isChecking = false; });
  };

  window.onboardingMinimize = function() {
    document.getElementById('onboardingExpanded').classList.add('hidden');
    document.getElementById('onboardingMinimized').classList.remove('hidden');
  };

  window.onboardingExpand = function(e) {
    if (e) e.stopPropagation();
    document.getElementById('onboardingMinimized').classList.add('hidden');
    document.getElementById('onboardingExpanded').classList.remove('hidden');
  };

  window.onboardingClose = function() {
    var panel = document.getElementById('onboardingPanel');
    panel.style.opacity = '0';
    setTimeout(function() { panel.style.display = 'none'; }, 500);
  };

  window.onboardingSkip = function() {
    if (typeof showNotification !== 'undefined') {
      showNotification('You can always revisit the guide at /guide', 'info');
    }
    csrfFetch('/business/onboarding/skip', { method: 'POST' })
    .then(function(r) { return r.json(); })
    .then(function(data) {
      if (data.completed) {
        window.onboardingClose();
      }
    })
    .catch(function() {});
  };

  window.onboardingCopyLink = function() {
    var panel = document.getElementById('onboardingPanel');
    var slug = panel ? panel.getAttribute('data-slug') : '';
    var text = window.location.origin + '/b/' + slug;
    if (navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard.writeText(text).then(function() {
        if (typeof showNotification !== 'undefined') {
          showNotification('Profile link copied! Share it with your clients.', 'success');
        }
      }).catch(function() {
        fallbackCopy(text);
      });
    } else {
      fallbackCopy(text);
    }
  };

  function fallbackCopy(text) {
    var ta = document.createElement('textarea');
    ta.value = text;
    ta.style.position = 'fixed';
    ta.style.opacity = '0';
    document.body.appendChild(ta);
    ta.select();
    try { document.execCommand('copy'); } catch (e) {}
    document.body.removeChild(ta);
    if (typeof showNotification !== 'undefined') {
      showNotification('Profile link copied!', 'success');
    }
  }

  window.onboardingOpenProfile = function() {
    if (typeof showBusinessProfile !== 'undefined') {
      showBusinessProfile();
    } else {
      window.location.href = '/business/dashboard';
    }
  };

  window.onboardingStartOrder = function() {
    var panel = document.getElementById('onboardingPanel');
    if (!panel) return;
    var clientId = panel.getAttribute('data-first-client-id');
    var convId = panel.getAttribute('data-first-conv-id');
    if (!clientId || !convId) {
      if (typeof showNotification !== 'undefined') {
        showNotification('No clients yet — share your profile link first!', 'warning');
      }
      return;
    }
    // Close the onboarding panel
    window.onboardingClose();
    // Load the client chat
    if (typeof loadClient !== 'undefined') {
      loadClient(clientId);
    }
    // Open the product picker
    setTimeout(function() {
      if (typeof showProductPicker !== 'undefined') {
        showProductPicker('business', convId, null, clientId);
      } else {
        window.location.href = '/business';
      }
    }, 300);
  };

  window.onboardingStartBooking = function() {
    var panel = document.getElementById('onboardingPanel');
    if (!panel) return;
    var clientId = panel.getAttribute('data-first-client-id');
    var convId = panel.getAttribute('data-first-conv-id');
    if (!clientId || !convId) {
      if (typeof showNotification !== 'undefined') {
        showNotification('No clients yet — share your profile link first!', 'warning');
      }
      return;
    }
    window.onboardingClose();
    if (typeof loadClient !== 'undefined') {
      loadClient(clientId);
    }
    setTimeout(function() {
      if (typeof openServicePicker !== 'undefined') {
        openServicePicker(clientId);
      } else {
        window.location.href = '/business';
      }
    }, 300);
  };

  window.onboardingNavigate = function() {
    // Panel stays open during navigation — page reload will refresh state
  };

  function pollOnboarding() {
    var panel = document.getElementById('onboardingPanel');
    if (!panel || panel.style.display === 'none') {
      if (pollInterval) clearInterval(pollInterval);
      return;
    }
    fetch('/business/onboarding/status?' + Date.now(), {
      headers: { 'Accept': 'application/json' }
    })
    .then(function(r) { return r.json(); })
    .then(function(data) {
      if (data.completed) {
        showStepCompleteAndAdvance();
        if (pollInterval) clearInterval(pollInterval);
      } else if (data.step && data.step !== undefined) {
        var currentStepEl = document.querySelector('.onboarding-step:not(.hidden)');
        if (currentStepEl && currentStepEl.id !== 'onboardingStep' + data.step) {
          window.location.reload();
        }
      }
    })
    .catch(function() {});
  }

  // Start polling on page load
  document.addEventListener('DOMContentLoaded', function() {
    var panel = document.getElementById('onboardingPanel');
    if (panel) {
      pollInterval = setInterval(pollOnboarding, 5000);
    }
  });

})();
