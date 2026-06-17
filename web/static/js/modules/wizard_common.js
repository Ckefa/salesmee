// Shared date utilities for product_picker and service_picker wizards
function formatDateDDMMYY(date) {
  var d = date.getDate().toString().padStart(2, '0');
  var m = (date.getMonth() + 1).toString().padStart(2, '0');
  var y = date.getFullYear().toString().slice(-2);
  return d + '/' + m + '/' + y;
}

function isTodayOrFuture(date) {
  var today = new Date();
  today.setHours(0, 0, 0, 0);
  return date >= today;
}

function syncDateDisplay(dateInput, displayId) {
  var display = document.getElementById(displayId);
  if (!display) return;
  if (dateInput.value) {
    var date = new Date(dateInput.value + 'T00:00:00');
    if (!isNaN(date.getTime())) {
      display.value = formatDateDDMMYY(date);
    }
  } else {
    display.value = '';
  }
}
