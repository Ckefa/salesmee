// Product Picker v2 - Multi-item ordering experience
let pickerProducts = [];
let pickerMode = 'business';
let pickerConversationId = null;
let pickerBusinessId = null;
let pickerClientId = null;
let pickerCart = [];
let pickerCurrentStep = 1;

// Edit mode state
let pickerEditMode = false;
let pickerGoBackToChat = false;
let pickerEditOrderId = null;
let pickerEditItems = [];
let pickerEditNotes = '';
let pickerEditAddress = '';

function showProductPicker(mode, convId, bizId, clientId, editOrderId, editItems, editNotes, editAddress) {
  pickerMode = mode || 'business';
  pickerConversationId = convId;
  pickerBusinessId = bizId;
  pickerClientId = clientId;
  pickerCart = [];
  pickerCurrentStep = 1;

  // Edit mode setup
  pickerEditMode = !!editOrderId;
  pickerEditOrderId = editOrderId || null;
  pickerEditItems = editItems || [];
  pickerEditNotes = editNotes || '';
  pickerEditAddress = editAddress || '';

  const modal = document.getElementById('productPickerModal');
  if (!modal) { showNotification('Product picker not loaded yet', 'error'); return; }

  const hintEl = document.getElementById('pickerSubmitHint');
  if (hintEl) {
    hintEl.textContent = mode === 'client'
      ? 'Your order will be sent to the business for review'
      : 'The order will be sent as a draft to the client';
  }
  const btnText = document.getElementById('pickerSubmitBtnText');
  if (btnText) {
    btnText.textContent = pickerEditMode ? 'Update Order'
      : (mode === 'client' ? 'Place Order' : 'Create Order');
  }

  modal.classList.remove('hidden');
  document.body.style.overflow = 'hidden';
  pickerGoToStep(1);
  loadPickerProducts();
}

function hideProductPicker() {
  const modal = document.getElementById('productPickerModal');
  if (modal) modal.classList.add('hidden');
  document.body.style.overflow = '';
  pickerCart = [];
  pickerProducts = [];
  pickerCurrentStep = 1;
  pickerEditMode = false;
  pickerEditOrderId = null;
  pickerEditItems = [];
  pickerEditNotes = '';
  pickerEditAddress = '';
}

function pickerGoToStep(step) {
  pickerCurrentStep = step;
  ['pickerStep1', 'pickerStep2', 'pickerStep3'].forEach((id, i) => {
    const el = document.getElementById(id);
    if (el) el.classList.toggle('hidden', i + 1 !== step);
  });

  const indicators = [
    { ind: 'step1Indicator', label: 'step1Label', line: 'step1to2' },
    { ind: 'step2Indicator', label: 'step2Label', line: 'step2to3' },
    { ind: 'step3Indicator', label: 'step3Label', line: null }
  ];
  indicators.forEach((item, i) => {
    const idx = i + 1;
    const indEl = document.getElementById(item.ind);
    const labelEl = document.getElementById(item.label);
    const isActive = idx <= step;
    if (indEl) {
      indEl.className = `step-indicator w-7 h-7 rounded-full flex items-center justify-center text-xs font-bold transition-colors duration-300 ${isActive ? 'bg-[var(--color-primary)] text-white' : 'bg-[var(--color-surface-tertiary)] text-[var(--color-text-muted)]'}`;
      if (isActive && idx < step) indEl.innerHTML = '<i class="fas fa-check text-[10px]"></i>';
      else indEl.textContent = String(idx);
    }
    if (labelEl) {
      labelEl.className = `text-xs font-medium transition-colors duration-300 ${isActive ? 'text-[var(--color-primary)]' : 'text-[var(--color-text-muted)]'}`;
    }
    if (item.line) {
      const lineEl = document.getElementById(item.line);
      if (lineEl) lineEl.className = `w-12 h-px mx-2 transition-colors duration-300 ${isActive ? 'bg-[var(--color-primary)]' : 'bg-[var(--color-border)]'}`;
    }
  });

  if (step === 2) pickerRenderCart();
  if (step === 3) pickerRenderCheckout();
  pickerUpdateCartBar();
}

// ========== CART OPERATIONS ==========

function pickerAddToCart(product, qty) {
  const existing = pickerCart.find(item => item.product.id === product.id);
  if (existing) {
    const newQty = Math.min(existing.quantity + qty, product.stock);
    existing.quantity = newQty;
  } else {
    pickerCart.push({ product, quantity: Math.min(qty, product.stock) });
  }
  pickerUpdateCartBar();
  pickerRenderProducts();
  showNotification(`${product.name} added to order`, 'success');

  const notice = document.getElementById('pickerCartNotice');
  if (notice && pickerCurrentStep === 1) {
    notice.classList.remove('picker-notice-pulse');
    void notice.offsetWidth;
    notice.classList.add('picker-notice-pulse');
    notice.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
  }
}

function pickerRemoveFromCart(productId) {
  pickerCart = pickerCart.filter(item => item.product.id !== productId);
  pickerUpdateCartBar();
  pickerRenderCart();
  pickerRenderCheckout();
}

function pickerUpdateCartItemQty(productId, newQty) {
  const item = pickerCart.find(i => i.product.id === productId);
  if (!item) return;
  if (newQty <= 0) {
    pickerRemoveFromCart(productId);
    return;
  }
  item.quantity = Math.min(newQty, item.product.stock);
  pickerUpdateCartBar();
  pickerRenderCart();
  pickerRenderCheckout();
}

function pickerGetCartCount() {
  return pickerCart.reduce((sum, item) => sum + item.quantity, 0);
}

function pickerGetCartTotal() {
  return pickerCart.reduce((sum, item) => sum + item.quantity * item.product.price, 0);
}

function pickerUpdateCartBar() {
  const bar = document.getElementById('pickerCartBar');
  const badge = document.getElementById('pickerCartBadge');
  const label = document.getElementById('pickerCartItemsLabel');
  const totalEl = document.getElementById('pickerCartTotal');
  const reviewBtn = document.getElementById('pickerReviewBtn');
  const notice = document.getElementById('pickerCartNotice');
  const noticeCount = document.getElementById('pickerNoticeCount');
  const noticePlural = document.getElementById('pickerNoticePlural');
  const noticeTotal = document.getElementById('pickerNoticeTotal');
  const count = pickerGetCartCount();

  if (count > 0) {
    if (bar) { bar.classList.remove('hidden'); bar.classList.add('flex'); }
    if (badge) { badge.textContent = count; badge.classList.remove('hidden'); }
    if (label) label.textContent = `${count} item${count !== 1 ? 's' : ''}`;
    if (totalEl) totalEl.textContent = pickerGetCartTotal().toFixed(2);
    if (reviewBtn) reviewBtn.disabled = false;

    if (notice) notice.classList.remove('hidden');
    if (noticeCount) noticeCount.textContent = count;
    if (noticePlural) noticePlural.textContent = count !== 1 ? 's' : '';
    if (noticeTotal) noticeTotal.textContent = pickerGetCartTotal().toFixed(2);
  } else {
    if (bar) { bar.classList.add('hidden'); bar.classList.remove('flex'); }
    if (reviewBtn) reviewBtn.disabled = true;
    if (notice) notice.classList.add('hidden');
  }
}

// ========== PRODUCT LOADING ==========

async function loadPickerProducts() {
  const grid = document.getElementById('pickerProductsGrid');
  if (!grid) return;
  grid.innerHTML = '<div class="text-center py-16 text-[var(--color-text-muted)]"><i class="fas fa-spinner fa-spin text-3xl mb-3"></i><p class="text-sm">Loading products...</p></div>';

  try {
    let url;
    if (pickerMode === 'client') {
      url = `/client/businesses/${pickerBusinessId}/products`;
    } else {
      url = `/business/conversations/${pickerConversationId}/products`;
    }
    const resp = await fetch(url, { credentials: 'same-origin', redirect: 'error' });

    if (!resp.ok) {
      const msg = resp.status === 401 ? 'Not authenticated. Please refresh.' :
                  resp.status === 403 ? 'Access denied.' :
                  resp.status === 404 ? 'Conversation not found.' :
                  `Server error (${resp.status})`;
      grid.innerHTML = `<div class="text-center py-12 text-[var(--color-warning)]"><i class="fas fa-exclamation-triangle text-3xl mb-3"></i><p>${msg}</p></div>`;
      return;
    }

    const data = await resp.json();
    let products;
    if (Array.isArray(data)) {
      products = data;
    } else if (data.products) {
      products = data.products;
    } else {
      products = [];
    }

    pickerProducts = products;
    const countEl = document.getElementById('pickerProductCount');
    if (countEl) countEl.textContent = `${products.length} product${products.length !== 1 ? 's' : ''} available`;
    pickerRenderProducts();
    if (pickerEditMode && pickerEditItems.length > 0) {
      pickerPrepopulateFromEditItems();
    }
  } catch (e) {
    console.error('Picker: Failed to load products:', e);
    grid.innerHTML = '<div class="text-center py-12 text-[var(--color-error)]"><i class="fas fa-exclamation-triangle text-3xl mb-3"></i><p>Failed to load products</p></div>';
  }
}

function pickerRenderProducts() {
  const grid = document.getElementById('pickerProductsGrid');
  if (!grid) return;

  const searchInput = document.getElementById('pickerSearch');
  const search = searchInput ? (searchInput.value || '').toLowerCase() : '';

  const clearBtn = document.getElementById('pickerSearchClear');
  if (clearBtn) clearBtn.classList.toggle('hidden', !search);

  if (!pickerProducts || pickerProducts.length === 0) {
    grid.innerHTML = '<div class="text-center py-16 text-[var(--color-text-muted)]"><i class="fas fa-box-open text-5xl mb-3"></i><p class="text-sm font-medium">No products available</p><p class="text-xs mt-1">Add products from the Business dashboard</p></div>';
    return;
  }

  const filtered = pickerProducts.filter(p => {
    const name = (p.name || p.Name || '').toLowerCase();
    const sku = (p.sku || p.SKU || '').toLowerCase();
    return !search || name.includes(search) || sku.includes(search);
  });

  if (filtered.length === 0) {
    grid.innerHTML = '<div class="text-center py-16 text-[var(--color-text-muted)]"><i class="fas fa-search text-4xl mb-3"></i><p class="text-sm">No products match your search</p></div>';
    return;
  }

  const cartIds = new Set(pickerCart.map(item => item.product.id));

  let html = '<div class="grid grid-cols-1 sm:grid-cols-2 gap-3">';
  filtered.forEach(p => {
    const prodId = p.id || p.ID;
    const name = p.name || p.Name || 'Product';
    const description = p.description || p.Description || '';
    const price = parseFloat(p.price || p.Price || 0);
    const stock = p.stock || p.Stock || 0;
    const minStock = p.min_stock || p.MinStock || 0;
    const sku = p.sku || p.SKU || '';
    const imgUrl = p.image_url || p.ImageURL || '';
    const inCart = cartIds.has(prodId);
    const cartItem = pickerCart.find(i => i.product.id === prodId);
    const cartQty = cartItem ? cartItem.quantity : 0;
    const outOfStock = stock <= 0;

    let stockColor = 'bg-[var(--color-success)]';
    let stockLabel = `${stock} in stock`;
    if (stock <= 0) { stockColor = 'bg-[var(--color-error)]'; stockLabel = 'Out of stock'; }
    else if (stock <= minStock) { stockColor = 'bg-[var(--color-error)]'; stockLabel = `Only ${stock} left - reorder soon`; }
    else if (stock <= 5) { stockColor = 'bg-[var(--color-warning)]'; stockLabel = `Only ${stock} left`; }

    const stockPct = stock > 0 ? Math.min((stock / Math.max(stock, 50)) * 100, 100) : 0;

    html += `
      <div class="product-picker-item border-2 rounded-xl transition-all duration-200 bg-[var(--color-surface)] overflow-hidden ${outOfStock ? 'border-[var(--color-error-light)] opacity-70' : inCart ? 'border-[var(--color-primary)] shadow-md shadow-[var(--color-primary)]/20' : 'border-[var(--color-border)] hover:border-[var(--color-primary)] hover:shadow-sm'}">
        <div class="flex p-3 gap-3">
          <!-- Image -->
          <div class="w-20 h-20 rounded-lg flex-shrink-0 overflow-hidden bg-[var(--color-surface-secondary)] border border-[var(--color-border-light)] relative">
            ${imgUrl ? `<img src="${imgUrl}" class="w-full h-full object-cover">` :
              `<div class="w-full h-full flex items-center justify-center text-2xl text-[var(--color-text-muted)]"><i class="fas fa-box"></i></div>`}
            ${outOfStock ? `<div class="absolute inset-0 bg-[var(--color-surface)]/60 flex items-center justify-center"><span class="text-[var(--color-error)] text-[10px] font-bold bg-[var(--color-error-light)] px-1.5 py-0.5 rounded">UNAVAILABLE</span></div>` : ''}
            ${inCart && !outOfStock ? `<div class="absolute top-1 right-1 bg-[var(--color-primary)] text-white text-[10px] font-bold w-5 h-5 rounded-full flex items-center justify-center shadow-sm">${cartQty}</div>` : ''}
          </div>
          <!-- Info -->
          <div class="flex-1 min-w-0">
            <div class="flex items-start justify-between gap-1">
              <h5 class="font-semibold text-sm text-[var(--color-text)] truncate">${name}</h5>
              <span class="text-base font-extrabold text-[var(--color-primary-dark)] flex-shrink-0 ml-1">$${price.toFixed(2)}</span>
            </div>
            ${sku ? `<span class="text-[10px] font-mono text-[var(--color-text-muted)] bg-[var(--color-surface-secondary)] px-1.5 py-0.5 rounded inline-block mt-0.5">SKU: ${sku}</span>` : ''}
            ${description ? `<p class="text-xs text-[var(--color-text-muted)] mt-1 line-clamp-2 leading-relaxed">${description}</p>` : ''}
            <!-- Stock bar -->
            ${stock > 0 ? `
            <div class="mt-1.5 flex items-center gap-1.5">
              <div class="flex-1 h-1.5 bg-[var(--color-surface-tertiary)] rounded-full overflow-hidden">
                <div class="h-full rounded-full ${stockColor}" style="width: ${stockPct}%"></div>
              </div>
              <span class="text-[10px] text-[var(--color-text-muted)] flex-shrink-0">${stock}</span>
            </div>` : `
            <div class="mt-1.5 text-[10px] text-[var(--color-error)] font-medium">Out of stock</div>`}
          </div>
        </div>
        <!-- Action -->
        <div class="px-3 pb-3">
          ${outOfStock ? `
          <div class="w-full py-2 rounded-lg bg-[var(--color-surface-tertiary)] text-[var(--color-text-muted)] text-xs font-medium text-center cursor-not-allowed">Unavailable</div>` : inCart ? `
          <div class="flex items-center gap-1">
            <button onclick="pickerQuickUpdate('${prodId}', ${cartQty - 1})" class="flex-1 py-2 rounded-lg border border-[var(--color-primary-light)] text-[var(--color-primary)] hover:bg-[var(--color-primary-light)] text-xs font-semibold transition active:scale-95"><i class="fas fa-minus"></i></button>
            <span class="w-10 text-center text-sm font-bold text-[var(--color-primary-dark)]">${cartQty}</span>
            <button onclick="pickerQuickUpdate('${prodId}', ${cartQty + 1})" class="flex-1 py-2 rounded-lg bg-[var(--color-primary)] text-white hover:bg-[var(--color-primary-hover)] text-xs font-semibold transition active:scale-95" ${cartQty >= stock ? 'disabled' : ''}><i class="fas fa-plus"></i></button>
          </div>` : `
          <button onclick="pickerAddToCartFromGrid(${prodId})" class="w-full py-2 rounded-lg bg-gradient-to-r from-[var(--color-primary)] to-[var(--color-primary-dark)] text-white hover:opacity-90 text-xs font-semibold transition active:scale-[0.98] shadow-sm">
            <i class="fas fa-plus mr-1"></i> Add to Order
          </button>`}
        </div>
      </div>`;
  });
  html += '</div>';
  grid.innerHTML = html;
}

function filterPickerProducts() {
  pickerRenderProducts();
}

function pickerQuickUpdate(prodId, newQty) {
  prodId = parseInt(prodId);
  if (newQty <= 0) {
    pickerRemoveFromCart(prodId);
    showNotification('Item removed from order', 'info');
  } else {
    pickerUpdateCartItemQty(prodId, newQty);
  }
  pickerRenderProducts();
}

function pickerAddToCartFromGrid(prodId) {
  const product = pickerProducts.find(p => (p.id || p.ID) === prodId);
  if (!product) return;
  const productObj = {
    id: product.id || product.ID,
    name: product.name || product.Name || 'Product',
    price: parseFloat(product.price || product.Price || 0),
    stock: product.stock || product.Stock || 0,
    sku: product.sku || product.SKU || '',
    description: product.description || product.Description || '',
    imgUrl: product.image_url || product.ImageURL || ''
  };
  pickerAddToCart(productObj, 1);
  pickerRenderProducts();
}

// ========== STEP 2: CART REVIEW ==========

function pickerRenderCart() {
  const container = document.getElementById('pickerCartItems');
  const summary = document.getElementById('pickerCartSummary');
  const countEl = document.getElementById('pickerReviewCount');
  const toDetailsBtn = document.getElementById('pickerToDetailsBtn');
  const subtotalItems = document.getElementById('pickerSubtotalItems');
  const subtotalAmount = document.getElementById('pickerSubtotalAmount');
  const totalEl = document.getElementById('pickerCartTotal');

  if (!container) return;

  const count = pickerGetCartCount();
  if (countEl) countEl.textContent = `${count} item${count !== 1 ? 's' : ''}`;
  if (totalEl) totalEl.textContent = pickerGetCartTotal().toFixed(2);

  if (pickerCart.length === 0) {
    container.innerHTML = '<div class="text-center py-12 text-[var(--color-text-muted)]"><i class="fas fa-shopping-cart text-4xl mb-3"></i><p class="text-sm">Your cart is empty</p><p class="text-xs mt-1">Go back and add some products</p></div>';
    if (summary) summary.classList.add('hidden');
    if (toDetailsBtn) toDetailsBtn.disabled = true;
    if (subtotalItems) subtotalItems.textContent = '0';
    if (subtotalAmount) subtotalAmount.textContent = '0.00';
    return;
  }

  if (toDetailsBtn) toDetailsBtn.disabled = false;
  if (subtotalItems) subtotalItems.textContent = count;
  if (subtotalAmount) subtotalAmount.textContent = pickerGetCartTotal().toFixed(2);

  let html = '';
  pickerCart.forEach(item => {
    const lineTotal = item.quantity * item.product.price;
    html += `
      <div class="flex items-center gap-3 p-3 bg-[var(--color-surface)] border border-[var(--color-border-light)] rounded-xl hover:border-[var(--color-border)] transition cart-item" data-cart-item-id="${item.product.id}">
        <div class="w-14 h-14 rounded-lg flex-shrink-0 overflow-hidden bg-[var(--color-surface-secondary)] border border-[var(--color-border-light)] flex items-center justify-center text-xl text-[var(--color-text-muted)]">
          ${item.product.imgUrl ? `<img src="${item.product.imgUrl}" class="w-full h-full object-cover">` : '<i class="fas fa-box"></i>'}
        </div>
        <div class="flex-1 min-w-0">
          <p class="font-semibold text-sm text-[var(--color-text)] truncate">${item.product.name}</p>
          ${item.product.sku ? `<span class="text-[10px] font-mono text-[var(--color-text-muted)]">SKU: ${item.product.sku}</span>` : ''}
          <p class="text-xs text-[var(--color-text-muted)] mt-0.5">${
            item.product.price.toFixed(2)} each</p>
        </div>
        <div class="flex items-center gap-2 flex-shrink-0">
          <div class="flex items-center border border-[var(--color-border)] rounded-lg overflow-hidden bg-[var(--color-surface)]">
            <button onclick="pickerCartDecrement(${item.product.id})" class="w-8 h-8 flex items-center justify-center text-[var(--color-text-secondary)] hover:bg-[var(--color-surface-secondary)] hover:text-[var(--color-primary)] text-xs transition ${item.quantity <= 1 ? 'text-[var(--color-error)] hover:text-[var(--color-error)]' : ''}">
              <i class="fas fa-${item.quantity <= 1 ? 'trash-alt' : 'minus'}"></i>
            </button>
            <span class="w-8 h-8 flex items-center justify-center text-sm font-bold text-[var(--color-text)] border-x border-[var(--color-border)] bg-[var(--color-surface-secondary)]">${item.quantity}</span>
            <button onclick="pickerCartIncrement(${item.product.id})" class="w-8 h-8 flex items-center justify-center text-[var(--color-text-secondary)] hover:bg-[var(--color-surface-secondary)] hover:text-[var(--color-primary)] text-xs transition" ${item.quantity >= item.product.stock ? 'disabled style="opacity:30;cursor:not-allowed"' : ''}>
              <i class="fas fa-plus"></i>
            </button>
          </div>
          <span class="text-sm font-bold text-[var(--color-text)] w-16 text-right">$${lineTotal.toFixed(2)}</span>
          <button onclick="pickerRemoveFromCart(${item.product.id})" class="text-[var(--color-text-muted)] hover:text-[var(--color-error)] text-xs p-1 transition" title="Remove">
            <i class="fas fa-times"></i>
          </button>
        </div>
      </div>`;
  });

  container.innerHTML = html;
  if (summary) summary.classList.remove('hidden');
}

function pickerCartIncrement(prodId) {
  const item = pickerCart.find(i => i.product.id === prodId);
  if (item && item.quantity < item.product.stock) {
    pickerUpdateCartItemQty(prodId, item.quantity + 1);
    pickerRenderProducts();
  }
}

function pickerCartDecrement(prodId) {
  const item = pickerCart.find(i => i.product.id === prodId);
  if (!item) return;
  if (item.quantity <= 1) {
    pickerRemoveFromCart(prodId);
    pickerRenderProducts();
  } else {
    pickerUpdateCartItemQty(prodId, item.quantity - 1);
    pickerRenderProducts();
  }
}

// ========== STEP 3: CHECKOUT ==========

function pickerRenderCheckout() {
  const container = document.getElementById('pickerCheckoutItems');
  const totalEl = document.getElementById('pickerCheckoutTotal');
  const countEl = document.getElementById('pickerCheckoutCount');
  const count = pickerGetCartCount();
  if (countEl) countEl.textContent = `${count} item${count !== 1 ? 's' : ''}`;
  if (totalEl) totalEl.textContent = pickerGetCartTotal().toFixed(2);
  if (!container) return;

  let html = '';
  pickerCart.forEach(item => {
    html += `<div class="flex items-center justify-between text-xs">
      <span class="truncate max-w-[200px]">${item.quantity}x ${item.product.name}</span>
      <span class="font-medium text-[var(--color-text)]">$${(item.quantity * item.product.price).toFixed(2)}</span>
    </div>`;
  });
  container.innerHTML = html;

  // Set default delivery date to tomorrow
  const dateInput = document.getElementById('pickerDeliveryDate');
  if (dateInput && !dateInput.value) {
    const tomorrow = new Date();
    tomorrow.setDate(tomorrow.getDate() + 1);
    dateInput.value = tomorrow.toISOString().split('T')[0];
  }
}

// ========== EDIT MODE ==========

function pickerPrepopulateFromEditItems() {
  pickerEditItems.forEach(editItem => {
    const product = pickerProducts.find(p => (p.id || p.ID) === editItem.product_id);
    if (!product) return;
    const productObj = {
      id: product.id || product.ID,
      name: product.name || product.Name || 'Product',
      price: parseFloat(product.price || product.Price || 0),
      stock: product.stock || product.Stock || 0,
      sku: product.sku || product.SKU || '',
      description: product.description || product.Description || '',
      imgUrl: product.image_url || product.ImageURL || ''
    };
    pickerAddToCart(productObj, editItem.quantity);
  });

  // Pre-fill step 3 fields
  if (pickerEditAddress) {
    const addrInput = document.getElementById('pickerDeliveryAddress');
    if (addrInput) addrInput.value = pickerEditAddress;
  }
  if (pickerEditNotes) {
    const notesInput = document.getElementById('pickerNotes');
    if (notesInput) notesInput.value = pickerEditNotes;
  }

  // Show notification
  showNotification('Cart pre-populated from existing order', 'info');
}

function openBusinessEditOrderPicker(orderId) {
  const card = document.querySelector(`[data-order-id="${orderId}"]`);
  if (!card) { showNotification('Order not found', 'error'); return; }

  if (!conversationId) { showNotification('No conversation found', 'error'); return; }
  if (!clientId) { showNotification('No client selected', 'error'); return; }

  // Extract items from DOM
  const items = [];
  card.querySelectorAll('[data-item-product-id]').forEach(el => {
    const pid = parseInt(el.dataset.itemProductId);
    const qtyText = el.querySelector('.text-sm.font-semibold')?.textContent || '1';
    const qty = parseInt(qtyText.replace('x', '')) || 1;
    if (pid) items.push({ product_id: pid, quantity: qty });
  });

  // Extract notes and address
  const notesEl = card.querySelector('.order-notes-data');
  let notes = notesEl ? notesEl.textContent.trim() : '';
  let address = '';

  // Parse delivery address from notes (stored as "📍 Delivery Address: ...")
  const addrMatch = notes.match(/📍 Delivery Address:\s*([^\n]*)/);
  if (addrMatch) {
    address = addrMatch[1].trim();
    notes = notes.replace(/📍 Delivery Address:\s*[^\n]*\n?/, '').trim();
  }

  // Strip delivery/contact prefixes added by the picker
  notes = notes.replace(/^📅 Delivery:.*\n?/m, '').trim();
  notes = notes.replace(/^📞 Contact:.*\n?/m, '').trim();

  showProductPicker('business', conversationId, null, clientId, orderId, items, notes, address);
}

function openClientEditOrderPicker(orderId) {
  const card = document.querySelector(`[data-order-id="${orderId}"]`);
  if (!card) { showNotification('Order not found', 'error'); return; }

  if (!businessId) { showNotification('No business selected', 'error'); return; }

  // Extract items from DOM
  const items = [];
  card.querySelectorAll('[data-item-product-id]').forEach(el => {
    const pid = parseInt(el.dataset.itemProductId);
    const qtyText = el.querySelector('.text-sm.font-semibold')?.textContent || '1';
    const qty = parseInt(qtyText.replace('x', '')) || 1;
    if (pid) items.push({ product_id: pid, quantity: qty });
  });

  const notesEl = card.querySelector('.order-notes-data');
  let notes = notesEl ? notesEl.textContent.trim() : '';
  let address = '';

  const addrMatch = notes.match(/📍 Delivery Address:\s*([^\n]*)/);
  if (addrMatch) {
    address = addrMatch[1].trim();
    notes = notes.replace(/📍 Delivery Address:\s*[^\n]*\n?/, '').trim();
  }
  notes = notes.replace(/^📅 Delivery:.*\n?/m, '').trim();
  notes = notes.replace(/^📞 Contact:.*\n?/m, '').trim();

  showProductPicker('client', null, businessId, clientId, orderId, items, notes, address);
}

// ========== SUBMIT ORDER ==========

async function submitProductOrder() {
  if (pickerCart.length === 0) {
    showNotification('Your cart is empty', 'error');
    return;
  }

  const deliveryAddress = document.getElementById('pickerDeliveryAddress').value.trim();
  if (!deliveryAddress) {
    showNotification('Please enter a delivery address', 'error');
    document.getElementById('pickerDeliveryAddress').focus();
    return;
  }

  const deliveryDate = document.getElementById('pickerDeliveryDate').value;
  const deliveryTimeSlot = document.getElementById('pickerDeliveryTimeSlot').value;
  const contactPhone = document.getElementById('pickerContactPhone').value.trim();
  const notes = document.getElementById('pickerNotes').value.trim();

  const submitBtn = document.getElementById('pickerSubmitOrderBtn');
  if (submitBtn) {
    submitBtn.disabled = true;
    submitBtn.innerHTML = '<i class="fas fa-spinner fa-spin mr-1"></i> Creating...';
  }

  const items = pickerCart.map(item => ({
    product_id: item.product.id,
    quantity: item.quantity
  }));

  let fullNotes = notes;
  if (deliveryDate) {
    const slotLabels = { morning: 'Morning (8AM-12PM)', afternoon: 'Afternoon (12PM-5PM)', evening: 'Evening (5PM-8PM)' };
    const slotLabel = slotLabels[deliveryTimeSlot] || 'Anytime';
    fullNotes = `📅 Delivery: ${deliveryDate} (${slotLabel})` + (fullNotes ? `\n${fullNotes}` : '');
  }
  if (contactPhone) {
    fullNotes = `📞 Contact: ${contactPhone}` + (fullNotes ? `\n${fullNotes}` : '');
  }

  try {
    let resp, data;

    if (pickerEditMode) {
      // ---------- EDIT MODE ----------
      const body = {
        items,
        notes: fullNotes,
        delivery_address: deliveryAddress
      };
      if (pickerMode === 'client') {
        resp = await fetch(`/client/orders/${pickerEditOrderId}/update`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': getCookie('csrf_token') },
          credentials: 'same-origin',
          body: JSON.stringify(body)
        });
      } else {
        resp = await fetch(`/business/orders/${pickerEditOrderId}`, {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': getCookie('csrf_token') },
          credentials: 'same-origin',
          body: JSON.stringify(body)
        });
      }
      data = await resp.json();
      if (data.success) {
        hideProductPicker();
        showNotification('Order updated successfully!', 'success');
        if (typeof fetchMessages === 'function') {
          setTimeout(() => fetchMessages(), 500);
        } else if (typeof startMessagePolling === 'function') {
          setTimeout(() => {
            fetch(`/client/businesses/${pickerBusinessId}/messages`)
              .then(r => r.text())
              .then(html => {
                const parser = new DOMParser();
                const doc = parser.parseFromString(html, 'text/html');
                const newMsgs = doc.getElementById('messages-container');
                const curMsgs = document.getElementById('messages-container');
                if (newMsgs && curMsgs) curMsgs.innerHTML = newMsgs.innerHTML;
              }).catch(console.error);
          }, 500);
        }
      } else {
        showNotification(data.error || 'Failed to update order', 'error');
      }
    } else if (pickerMode === 'client') {
      resp = await fetch('/client/orders', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': getCookie('csrf_token') },
        credentials: 'same-origin',
        body: JSON.stringify({
          business_id: parseInt(pickerBusinessId),
          items: items,
          delivery_address: deliveryAddress,
          notes: fullNotes
        })
      });
      data = await resp.json();
      if (data.success) {
        hideProductPicker();
        showNotification('Order placed successfully!', 'success');
        if (pickerGoBackToChat) {
          pickerGoBackToChat = false;
          setTimeout(function () {
            var btn = document.querySelector('[hx-get*="/messages"]');
            if (btn && btn.click) btn.click();
          }, 800);
        } else if (typeof startMessagePolling === 'function') {
          setTimeout(() => {
            fetch(`/client/businesses/${pickerBusinessId}/messages`)
              .then(r => r.text())
              .then(html => {
                const parser = new DOMParser();
                const doc = parser.parseFromString(html, 'text/html');
                const newMsgs = doc.getElementById('messages-container');
                const curMsgs = document.getElementById('messages-container');
                if (newMsgs && curMsgs) curMsgs.innerHTML = newMsgs.innerHTML;
              }).catch(console.error);
          }, 500);
        }
      } else {
        showNotification(data.error || 'Failed to place order', 'error');
      }
    } else {
      resp = await fetch(`/business/conversations/${pickerConversationId}/order-draft`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': getCookie('csrf_token') },
        credentials: 'same-origin',
        body: JSON.stringify({
          items,
          notes: fullNotes,
          delivery_address: deliveryAddress
        })
      });
      data = await resp.json();
      if (data.success) {
        hideProductPicker();
        showNotification('Order draft created successfully!', 'success');
        if (typeof fetchMessages === 'function') {
          setTimeout(() => fetchMessages(), 500);
        }
      } else {
        showNotification(data.error || 'Failed to create order', 'error');
      }
    }
  } catch (e) {
    console.error('Failed to create order:', e);
    showNotification('Failed to create order. Check console.', 'error');
  }

  if (submitBtn) {
    submitBtn.disabled = false;
    const btnLabel = pickerEditMode ? 'Update Order' :
      (pickerMode === 'client' ? 'Place Order' : 'Create Order');
    submitBtn.innerHTML = '<i class="fas fa-paper-plane mr-1"></i> ' + btnLabel;
  }
}

// ========== ENTRY POINTS ==========

function openBusinessProductPicker(clientId) {
  if (!clientId) { showNotification('No client selected', 'error'); return; }
  fetch(`/business/clients/${clientId}/conversation-id`, { credentials: 'same-origin' })
    .then(r => r.json())
    .then(data => {
      if (data.conversation_id) {
        showProductPicker('business', data.conversation_id, null, clientId);
      } else {
        showNotification('Could not find conversation', 'error');
      }
    })
    .catch(e => {
      console.error('Failed to get conversation:', e);
      showNotification('Failed to get conversation', 'error');
    });
}

function openClientProductPicker() {
  if (!businessId) { showNotification('No business selected', 'error'); return; }
  showProductPicker('client', null, businessId, clientId);
}

function openClientProductPickerWithProduct(productId) {
  if (!businessId) { showNotification('No business selected', 'error'); return; }
  pickerGoBackToChat = true;
  showProductPicker('client', null, businessId, clientId);

  const maxAttempts = 50;
  let attempts = 0;
  const interval = setInterval(function () {
    attempts++;
    if (pickerProducts.length > 0) {
      clearInterval(interval);
      var product = pickerProducts.find(function (p) { return p.id === productId; });
      if (product) {
        pickerAddToCart(product, 1);
        pickerGoToStep(2);
      }
    } else if (attempts >= maxAttempts) {
      clearInterval(interval);
    }
  }, 100);
}