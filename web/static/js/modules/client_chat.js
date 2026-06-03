

function openClientOrderModal() {
  loadClientProducts();
  document.getElementById('clientOrderModal').classList.remove('hidden');
}

function hideClientOrderModal() {
  document.getElementById('clientOrderModal').classList.add('hidden');
  document.getElementById('clientOrderForm').reset();
}

function openClientBookingModal() {
  loadClientServices();
  document.getElementById('clientBookingModal').classList.remove('hidden');
}

function hideClientBookingModal() {
  document.getElementById('clientBookingModal').classList.add('hidden');
  document.getElementById('clientBookingForm').reset();
}

async function loadClientProducts() {
  try {
    const response = await fetch(`/client/businesses/${businessId}/products`);
    const products = await response.json();
    const select = document.getElementById('clientOrderProduct');
    select.innerHTML = '<option value="">Choose a product...</option>';
    products.forEach(p => {
      const opt = document.createElement('option');
      opt.value = p.id;
      opt.textContent = `${p.name} - $${p.price}`;
      select.appendChild(opt);
    });
  } catch (error) {
    console.error('Error loading products:', error);
  }
}

async function loadClientServices() {
  try {
    const response = await fetch(`/client/businesses/${businessId}/services`);
    if (!response.ok) {
      if (response.status === 401) {
        showNotification('Please login to access services', 'error');
        return;
      }
      throw new Error(`HTTP error! status: ${response.status}`);
    }
    const services = await response.json();
    const select = document.getElementById('clientBookingService');
    select.innerHTML = '<option value="">Choose a service...</option>';
    if (services.length === 0) {
      select.innerHTML = '<option value="">No services available</option>';
      showNotification('No services available for booking', 'warning');
      return;
    }
    services.forEach(s => {
      const opt = document.createElement('option');
      opt.value = s.id;
      opt.textContent = `${s.name} - $${s.min_price || s.max_price || 'Price not set'}`;
      select.appendChild(opt);
    });
  } catch (error) {
    console.error('Error loading services:', error);
    showNotification('Failed to load services', 'error');
  }
}

function submitOrderForm() {
  const productSelect = document.getElementById('clientOrderProduct');
  const quantityInput = document.getElementById('clientOrderQuantity');
  const addressInput = document.getElementById('clientOrderAddress');
  const notesInput = document.getElementById('clientOrderNotes');

  if (!productSelect.value) return showNotification('Please select a product', 'error');
  if (!quantityInput.value || quantityInput.value < 1) return showNotification('Please enter a valid quantity', 'error');

  const data = {
    product_id: parseInt(productSelect.value),
    quantity: parseInt(quantityInput.value),
    delivery_address: addressInput.value,
    notes: notesInput.value,
    business_id: parseInt(businessId)
  };

  fetch('/client/orders', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': getCookie('csrf_token') },
    body: JSON.stringify(data)
  })
    .then(r => r.json())
    .then(data => {
      if (data.success) {
        hideClientOrderModal();
        if (data.order) {
          addOrderMessageToChat({ ...data.order, product_name: data.product_name, quantity: data.quantity });
        }
        showNotification('Order request sent successfully!', 'success');
      } else {
        showNotification(data.error || 'Failed to send order request', 'error');
      }
    })
    .catch(e => { console.error(e); showNotification('Failed to send order request', 'error'); });
}

function submitBookingForm() {
  const serviceSelect = document.getElementById('clientBookingService');
  const dateInput = document.getElementById('clientBookingDate');
  const timeInput = document.getElementById('clientBookingTime');
  const notesInput = document.getElementById('clientBookingNotes');

  if (!serviceSelect.value) return showNotification('Please select a service', 'error');
  if (!dateInput.value) return showNotification('Please select a date', 'error');
  if (!timeInput.value) return showNotification('Please select a time', 'error');

  const bookingDateTime = `${dateInput.value}T${timeInput.value}:00Z`;

  fetch('/client/bookings', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': getCookie('csrf_token') },
    body: JSON.stringify({
      service_id: parseInt(serviceSelect.value),
      scheduled_date: bookingDateTime,
      notes: notesInput.value,
      business_id: businessId
    })
  })
    .then(r => r.json())
    .then(data => {
      if (data.success) {
        hideClientBookingModal();
        if (data.booking) {
          addBookingMessageToChat({ ...data.booking, service_name: data.service_name });
        }
        showNotification('Booking request sent successfully!', 'success');
      } else {
        showNotification(data.error || 'Failed to send booking request', 'error');
      }
    })
    .catch(e => { console.error(e); showNotification('Failed to send booking request', 'error'); });
}

scrollToBottom();
markAsRead();
startMessagePolling();

function scrollToBottom() {
  var container = document.getElementById('messages-container');
  if (container) requestAnimationFrame(function () {
    container.scrollTop = container.scrollHeight;
  });
}

let pollingInterval = null;

function markAsRead() {
  fetch(`/client/businesses/${businessId}/read`, { method: 'PUT', headers: { 'X-CSRF-Token': getCookie('csrf_token') } })
    .then(function () {
      var badge = document.querySelector('.business-item[data-business-id="' + businessId + '"] .unread-badge');
      if (badge) badge.remove();
    })
    .catch(console.error);
}

function startMessagePolling() {
  pollingInterval = setInterval(function () {
    fetch(`/client/businesses/${businessId}/messages`)
      .then(r => r.text())
      .then(html => {
        const parser = new DOMParser();
        const doc = parser.parseFromString(html, 'text/html');
        const newMsgs = doc.getElementById('messages-container');
        const curMsgs = document.getElementById('messages-container');
        if (newMsgs && curMsgs && newMsgs.innerHTML !== curMsgs.innerHTML) {
          curMsgs.innerHTML = newMsgs.innerHTML;
          curMsgs.scrollTop = curMsgs.scrollHeight;
          markAsRead();
        }
      })
      .catch(console.error);
  }, 5000);
}

function addOrderMessageToChat(order) {
  const container = document.getElementById('messages-container');
  if (!container) return;
  const status = order.status || 'pending';
  const bgClass = status === 'pending' ? 'bg-[var(--color-warning-light)] border-[var(--color-warning)]' :
    status === 'client_confirmed' ? 'bg-[var(--color-info-light)] border-[var(--color-info)]' :
    status === 'confirmed' ? 'bg-[var(--color-info-light)] border-[var(--color-info)]' :
    status === 'fulfilled' || status === 'completed' ? 'bg-[var(--color-success-light)] border-[var(--color-success)]' :
    status === 'cancelled' ? 'bg-[var(--color-error-light)] border-[var(--color-error)]' : 'bg-[var(--color-info-light)] border-[var(--color-info)]';
  const iconColor = status === 'pending' ? 'text-[var(--color-warning)]' :
    status === 'client_confirmed' ? 'text-[var(--color-info)]' :
    status === 'confirmed' ? 'text-[var(--color-info)]' :
    status === 'fulfilled' || status === 'completed' ? 'text-[var(--color-success)]' :
    status === 'cancelled' ? 'text-[var(--color-error)]' : 'text-[var(--color-info)]';
  const statusLabel = status === 'pending' ? 'Pending' :
    status === 'client_confirmed' ? 'Confirmed' :
    status === 'confirmed' ? 'Confirmed' :
    status === 'fulfilled' || status === 'completed' ? 'Completed' :
    status === 'cancelled' ? 'Cancelled' : 'Pending';
  const statusBadgeBg = status === 'pending' ? 'bg-[var(--color-warning-light)] text-[var(--color-warning)]' :
    status === 'client_confirmed' ? 'bg-[var(--color-info-light)] text-[var(--color-info)]' :
    status === 'confirmed' ? 'bg-[var(--color-info-light)] text-[var(--color-info)]' :
    status === 'fulfilled' || status === 'completed' ? 'bg-[var(--color-success-light)] text-[var(--color-success)]' :
    status === 'cancelled' ? 'bg-[var(--color-error-light)] text-[var(--color-error)]' : 'bg-[var(--color-warning-light)] text-[var(--color-warning)]';
  const div = document.createElement('div');
  div.className = 'flex justify-end';
  div.innerHTML = `<div class="max-w-xs lg:max-w-md w-full">
    <div class="${bgClass} border rounded-lg px-4 py-3" data-message-id="${order.id}" data-order-id="${order.id}">
      <div class="flex items-center justify-between mb-2">
        <div class="flex items-center space-x-2">
          <i class="fas fa-shopping-cart ${iconColor}"></i>
          <span class="font-semibold ${iconColor} text-sm">[${order.id}]</span>
          <span class="text-[var(--color-text)] text-sm">${order.product_name || 'Product'}</span>
        </div>
        <button onclick="openClientEditOrderPicker(${order.id})" class="${iconColor} hover:opacity-80 text-xs" title="Edit Order">
          <i class="fas fa-edit"></i>
        </button>
      </div>
      <div class="order-details text-sm text-[var(--color-text)]">
        <p class="text-sm">Order #${order.order_number} - ${order.quantity || 1}x - $${parseFloat(order.total_amount).toFixed(2)}</p>
        <p class="hidden order-notes-data">${order.notes || ''}</p>
      </div>
      <div class="flex items-center justify-between mt-2">
        <p class="text-xs text-[var(--color-text-muted)]">${new Date().toLocaleTimeString('en-US', {hour:'numeric', minute:'2-digit'})}</p>
        <span class="text-xs ${statusBadgeBg} px-2 py-1 rounded">${statusLabel}</span>
      </div>
    </div>
  </div>`;
  container.appendChild(div);
  container.scrollTop = container.scrollHeight;
}

function addBookingMessageToChat(booking) {
  const container = document.getElementById('messages-container');
  if (!container) return;
  const bookingDate = new Date(booking.scheduled_date);
  const bookingNumber = booking.booking_number || booking.id;
  const serviceName = booking.service_name || '';
  const duration = booking.duration || '';
  const totalAmount = booking.total_amount || '';
  const notes = booking.notes || '';
  const status = booking.status || 'pending';
  const dateStr = bookingDate.toLocaleDateString('en-US', { weekday: 'short', month: 'short', day: 'numeric' });
  const timeStr = bookingDate.toLocaleTimeString('en-US', { hour: 'numeric', minute: '2-digit' });
  const statusClass = status === 'pending' ? 'bg-[var(--color-warning-light)] text-[var(--color-warning)]' :
    status === 'client_confirmed' ? 'bg-[var(--color-info-light)] text-[var(--color-info)]' :
    status === 'completed' ? 'bg-[var(--color-success-light)] text-[var(--color-success)]' :
    status === 'cancelled' ? 'bg-[var(--color-error-light)] text-[var(--color-error)]' : 'bg-[var(--color-surface-tertiary)] text-[var(--color-text-secondary)]';
  const borderClass = status === 'pending' ? 'border-[var(--color-warning)] bg-[var(--color-warning-light)]' :
    status === 'client_confirmed' ? 'border-[var(--color-info)] bg-[var(--color-info-light)]' :
    status === 'completed' ? 'border-[var(--color-success)] bg-[var(--color-success-light)]' :
    status === 'cancelled' ? 'border-[var(--color-error)] bg-[var(--color-error-light)]' : 'border-[var(--color-border)] bg-[var(--color-surface-secondary)]';
  const iconClass = status === 'pending' ? 'text-[var(--color-warning)]' :
    status === 'client_confirmed' ? 'text-[var(--color-info)]' :
    status === 'completed' ? 'text-[var(--color-success)]' :
    status === 'cancelled' ? 'text-[var(--color-error)]' : 'text-[var(--color-text-secondary)]';

  let extraHtml = '';
  if (status === 'pending') {
    extraHtml = '<div class="mt-2 pt-2 border-t border-[var(--color-border)]/50"><p class="text-xs text-center text-[var(--color-warning)] font-medium"><i class="fas fa-clock mr-1"></i>Awaiting business confirmation</p></div>';
  } else if (status === 'client_confirmed') {
    extraHtml = '<div class="mt-2 pt-2 border-t border-[var(--color-border)]/50"><p class="text-xs text-center text-[var(--color-info)] font-medium"><i class="fas fa-check-circle mr-1"></i>Your booking is confirmed</p></div>';
  } else if (status === 'completed') {
    extraHtml = '<div class="mt-2 pt-2 border-t border-[var(--color-border)]/50"><p class="text-xs text-center text-[var(--color-success)] font-medium"><i class="fas fa-check-double mr-1"></i>Service completed</p></div>';
  } else if (status === 'cancelled') {
    extraHtml = '<div class="mt-2 pt-2 border-t border-[var(--color-border)]/50"><p class="text-xs text-center text-[var(--color-error)] font-medium"><i class="fas fa-ban mr-1"></i>This booking was cancelled</p></div>';
  }

  container.insertAdjacentHTML('beforeend', `
    <div class="flex justify-end">
      <div class="max-w-xs lg:max-w-md w-full">
        <div class="${borderClass} border rounded-lg px-4 py-3" data-message-id="${booking.id}" data-booking-id="${booking.id}">
          <div class="flex items-center justify-between mb-2">
            <div class="flex items-center space-x-2 min-w-0">
              <i class="fas fa-calendar-check ${iconClass}"></i>
              <span class="font-semibold text-sm ${iconClass}">#${bookingNumber}</span>
              <span class="text-[var(--color-text)] text-sm truncate">${serviceName}</span>
            </div>
            <div class="flex items-center space-x-1 flex-shrink-0 ml-2">
              ${status === 'pending' ? '<button onclick="cancelBooking(' + booking.id + ')" class="text-[var(--color-error)] hover:opacity-80 text-xs" title="Cancel Booking"><i class="fas fa-times"></i></button>' : ''}
              <button onclick="openClientEditBookingPicker(${booking.id})" class="${iconClass} hover:opacity-80 text-xs" title="Edit Booking">
                <i class="fas fa-edit"></i>
              </button>
            </div>
          </div>
          <div class="booking-details text-sm text-[var(--color-text)] space-y-1">
            <p class="flex items-center space-x-1">
              <i class="fas fa-clock text-xs text-[var(--color-text-muted)]"></i>
              <span>${dateStr} at ${timeStr}</span>
            </p>
            ${duration ? '<p class="flex items-center space-x-1"><i class="fas fa-hourglass-half text-xs text-[var(--color-text-muted)]"></i><span>' + duration + ' min</span></p>' : ''}
            ${totalAmount ? '<p class="flex items-center space-x-1"><i class="fas fa-tag text-xs text-[var(--color-text-muted)]"></i><span>$' + parseFloat(totalAmount).toFixed(2) + '</span></p>' : ''}
            ${notes ? '<p class="text-xs text-[var(--color-text-muted)] italic mt-1 border-t border-[var(--color-border)] pt-1">' + notes + '</p>' : ''}
            <p class="hidden booking-notes-data">${notes}</p>
          </div>
          <div class="flex items-center justify-between mt-3 pt-2 border-t border-[var(--color-border)]/50">
            <p class="text-xs text-[var(--color-text-muted)]">Just now</p>
            <span class="text-xs font-medium ${statusClass} px-2 py-0.5 rounded-full booking-status">${status}</span>
          </div>
          ${extraHtml}
        </div>
      </div>
    </div>`);
  container.scrollTop = container.scrollHeight;
}

function clientConfirmOrder(orderId) {
  showConfirmModal({ title: 'Confirm Order', message: 'Confirm this order?' }).then(function(confirmed) {
    if (!confirmed) return;
    fetch(`/client/orders/${orderId}/confirm`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': getCookie('csrf_token') },
    body: JSON.stringify({ items: [] })
  })
    .then(r => r.json())
    .then(data => {
      if (data.success) {
        showNotification(data.message || 'Order confirmed!', 'success');
        // Trigger polling refresh
        setTimeout(() => {
          fetch(`/client/businesses/${businessId}/messages`)
            .then(r => r.text())
            .then(html => {
              const parser = new DOMParser();
              const doc = parser.parseFromString(html, 'text/html');
              const newMsgs = doc.getElementById('messages-container');
              const curMsgs = document.getElementById('messages-container');
              if (newMsgs && curMsgs) {
                curMsgs.innerHTML = newMsgs.innerHTML;
              }
            });
        }, 500);
      } else {
        showNotification(data.error || 'Failed to confirm order', 'error');
      }
    })
    .catch(e => { console.error(e); showNotification('Failed to confirm order', 'error'); });
  });
}

function clientOrderItemIncrement(orderId, productId, btn) {
  const qtySpan = btn.parentElement.querySelector('.qty-value');
  const current = parseInt(qtySpan.textContent);
  qtySpan.textContent = current + 1;
  updateClientOrderTotal(orderId);
}

function clientOrderItemDecrement(orderId, productId, btn) {
  const qtySpan = btn.parentElement.querySelector('.qty-value');
  const current = parseInt(qtySpan.textContent);
  if (current > 1) {
    qtySpan.textContent = current - 1;
  }
  updateClientOrderTotal(orderId);
}

function updateClientOrderTotal(orderId) {
  const card = document.querySelector(`[data-order-id="${orderId}"]`);
  if (!card) return;
  let total = 0;
  card.querySelectorAll('[data-item-product-id]').forEach(item => {
    const qty = parseInt(item.querySelector('.qty-value').textContent);
    const priceEl = item.closest('.flex.items-center.justify-between').querySelector('.text-sm.font-bold');
    const priceText = priceEl ? priceEl.textContent.replace(/[^0-9.]/g, '') : '0';
    total += qty * parseFloat(priceText);
  });
  const totalEl = card.querySelector('.text-lg.font-bold');
  if (totalEl) totalEl.textContent = (typeof currencySymbol !== 'undefined' ? currencySymbol : '$') + total.toFixed(2);
}

function cancelOrder(orderId) {
  showConfirmModal({ title: 'Cancel Order', message: 'Are you sure you want to cancel this order?', confirmClass: 'bg-[var(--color-error)] text-white', confirmText: 'Cancel Order' }).then(function(confirmed) {
    if (!confirmed) return;
    fetch(`/client/orders/${orderId}/cancel`, {
    method: 'POST',
    headers: { 'Authorization': 'Bearer ' + getCookie('client_token'), 'X-CSRF-Token': getCookie('csrf_token') }
  })
    .then(r => r.json())
    .then(data => {
      if (data.success) {
        showNotification('Order cancelled successfully', 'success');
      } else {
        showNotification(data.error || 'Failed to cancel order', 'error');
      }
    })
    .catch(e => { console.error(e); showNotification('Failed to cancel order', 'error'); });
  });
}

function cancelBooking(bookingId) {
  showConfirmModal({ title: 'Cancel Booking', message: 'Are you sure you want to cancel this booking?', confirmClass: 'bg-[var(--color-error)] text-white', confirmText: 'Cancel Booking' }).then(function(confirmed) {
    if (!confirmed) return;
  fetch(`/client/bookings/${bookingId}/cancel`, {
    method: 'POST',
    headers: { 'X-CSRF-Token': getCookie('csrf_token') }
  })
    .then(r => r.json())
    .then(data => {
      if (data.success) {
        showNotification('Booking cancelled successfully', 'success');
      } else {
        showNotification(data.error || 'Failed to cancel booking', 'error');
      }
    })
    .catch(e => { console.error(e); showNotification('Failed to cancel booking', 'error'); });
  });
}

function clientConfirmBooking(bookingId) {
  showConfirmModal({ title: 'Approve Booking', message: 'Are you sure you want to approve this booking?', confirmText: 'Approve', confirmClass: 'bg-[var(--color-success)] text-white' }).then(function(confirmed) {
    if (!confirmed) return;
  fetch(`/client/bookings/${bookingId}/confirm`, {
    method: 'POST',
    headers: { 'X-CSRF-Token': getCookie('csrf_token') }
  })
    .then(r => r.json())
    .then(data => {
      if (data.success) {
        showNotification('Booking confirmed!', 'success');
        if (typeof startMessagePolling === 'function') startMessagePolling();
      } else {
        showNotification(data.error || 'Failed to confirm booking', 'error');
      }
    })
    .catch(e => { console.error(e); showNotification('Failed to confirm booking', 'error'); });
  });
}

function toggleMediaTray() {
  var tray = document.getElementById('media-tray');
  var icon = document.getElementById('media-icon');
  if (tray) {
    tray.classList.toggle('hidden');
    if (icon) {
      icon.classList.toggle('fa-paperclip');
      icon.classList.toggle('fa-times');
    }
  }
}

function triggerMediaUpload(type) {
  var input = document.getElementById('media-input-' + type);
  if (input) input.click();
  var tray = document.getElementById('media-tray');
  if (tray && !tray.classList.contains('hidden')) {
    tray.classList.add('hidden');
    var icon = document.getElementById('media-icon');
    if (icon) icon.classList.replace('fa-times', 'fa-paperclip');
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
