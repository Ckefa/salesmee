let currentBusinessId = null;
let heartbeatInterval = null;

document.addEventListener('DOMContentLoaded', function () {
  document.addEventListener('click', function (e) {
    const item = e.target.closest('.business-item');
    if (item) {
      loadBusiness(item.getAttribute('data-business-id'));
    }
  });
  startHeartbeat();
});

function startHeartbeat() {
  heartbeatInterval = setInterval(function () {
    fetch('/client/heartbeat', {
      method: 'POST',
      headers: { 'Authorization': 'Bearer ' + getCookie('client_token') }
    }).catch(console.error);
  }, 30000);
}

function stopHeartbeat() {
  if (heartbeatInterval) {
    clearInterval(heartbeatInterval);
    heartbeatInterval = null;
  }
}

window.addEventListener('beforeunload', stopHeartbeat);

function loadBusiness(businessId) {
  currentBusinessId = businessId;
  document.querySelectorAll('.business-item').forEach(item => {
    item.classList.remove('bg-[var(--color-info-light)]', 'border-l-4', 'border-[var(--color-info)]');
  });
  const el = document.querySelector(`[data-business-id="${businessId}"]`);
  if (el) el.classList.add('bg-[var(--color-info-light)]', 'border-l-4', 'border-[var(--color-info)]');
  htmx.ajax('GET', `/client/businesses/${businessId}/messages`, {
    target: '#chat-area',
    swap: 'innerHTML'
  });
}

function sendMessage() {
  const form = document.getElementById('message-form');
  if (form) form.submit();
}

function disconnectBusiness(businessId) {
  if (!confirm('Remove this business from your list? You can reconnect later.')) return;
  fetch('/client/disconnect/' + businessId, { method: 'POST' })
    .then(r => r.json())
    .then(data => {
      if (data.success) {
        location.reload();
      } else {
        showNotification('Failed to remove business', 'error');
      }
    })
    .catch(() => showNotification('Failed to remove business', 'error'));
}

function hideClientOrderModal() {
  document.getElementById('clientOrderModal')?.classList.add('hidden');
  document.getElementById('clientOrderForm')?.reset();
}

function hideClientBookingModal() {
  document.getElementById('clientBookingModal')?.classList.add('hidden');
  document.getElementById('clientBookingForm')?.reset();
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
    headers: { 'Content-Type': 'application/json' },
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
