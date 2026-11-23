(function () {
  'use strict';

  var root = document.getElementById('stats');
  var errorBox = document.getElementById('stats-error');
  var palette = ['#2b3a55', '#e4572e', '#29a38b', '#f3a712', '#7b61c4', '#4c9be8', '#c2185b', '#8d9aab'];

  var catChart = null;
  var monthChart = null;

  function money(v) {
    return v.toLocaleString('ru-RU', { minimumFractionDigits: 2, maximumFractionDigits: 2 }) + ' ₽';
  }

  function showError(msg) {
    errorBox.textContent = msg;
    errorBox.hidden = false;
  }

  function clearError() {
    errorBox.hidden = true;
  }

  function getJSON(url) {
    return fetch(url, { headers: { Accept: 'application/json' }, credentials: 'same-origin' })
      .then(function (res) {
        if (res.status === 401) {
          window.location.href = '/login';
          throw new Error('unauthorized');
        }
        return res.json().then(function (data) {
          if (!res.ok) {
            throw new Error(data.error || 'Ошибка сервера');
          }
          return data;
        });
      });
  }

  function toggle(prefix, hasData) {
    document.getElementById(prefix + '-empty').hidden = hasData;
    document.getElementById(prefix + '-box').hidden = !hasData;
  }

  function loadCategories() {
    var params = new URLSearchParams();
    var from = document.getElementById('cat-from').value;
    var to = document.getElementById('cat-to').value;
    if (from) params.set('from', from);
    if (to) params.set('to', to);

    return getJSON('/api/stats/by-category?' + params.toString())
      .then(function (data) {
        clearError();
        var hasData = data.values.length > 0;
        toggle('cat', hasData);
        document.getElementById('cat-total').textContent = hasData ? 'Всего: ' + money(data.total) : '';

        if (catChart) catChart.destroy();
        if (!hasData) { catChart = null; return; }

        catChart = new Chart(document.getElementById('cat-chart'), {
          type: 'pie',
          data: {
            labels: data.labels,
            datasets: [{
              data: data.values,
              backgroundColor: data.labels.map(function (_, i) { return palette[i % palette.length]; })
            }]
          },
          options: {
            plugins: {
              legend: { position: 'bottom' },
              tooltip: { callbacks: { label: function (ctx) { return ctx.label + ': ' + money(ctx.parsed); } } }
            }
          }
        });
      })
      .catch(function (err) {
        if (err.message !== 'unauthorized') showError(err.message);
      });
  }

  function loadMonths() {
    var year = document.getElementById('month-year').value;

    return getJSON('/api/stats/by-month?year=' + encodeURIComponent(year))
      .then(function (data) {
        clearError();
        var hasData = data.total > 0;
        toggle('month', hasData);
        document.getElementById('month-total').textContent = hasData ? 'Всего за ' + data.year + ' год: ' + money(data.total) : '';

        if (monthChart) monthChart.destroy();
        if (!hasData) { monthChart = null; return; }

        monthChart = new Chart(document.getElementById('month-chart'), {
          type: 'bar',
          data: {
            labels: data.labels,
            datasets: [{ label: 'Сумма, ₽', data: data.values, backgroundColor: '#2b3a55' }]
          },
          options: {
            plugins: {
              legend: { display: false },
              tooltip: { callbacks: { label: function (ctx) { return money(ctx.parsed.y); } } }
            },
            scales: { y: { beginAtZero: true } }
          }
        });
      })
      .catch(function (err) {
        if (err.message !== 'unauthorized') showError(err.message);
      });
  }

  document.getElementById('cat-from').addEventListener('change', loadCategories);
  document.getElementById('cat-to').addEventListener('change', loadCategories);
  document.getElementById('cat-reset').addEventListener('click', function () {
    document.getElementById('cat-from').value = '';
    document.getElementById('cat-to').value = '';
    loadCategories();
  });
  document.getElementById('month-year').addEventListener('change', loadMonths);

  if (typeof Chart === 'undefined') {
    showError('Не удалось загрузить библиотеку Chart.js (проверьте подключение к интернету).');
    return;
  }

  loadCategories();
  loadMonths();
})();
