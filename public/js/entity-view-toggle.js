(function () {
  'use strict';

  document.querySelectorAll('[data-entity-view]').forEach(function (root) {
    const cards = root.querySelector('[data-view-cards]');
    const table = root.querySelector('[data-view-table]');
    const toolbar = root.querySelector('[data-view-toolbar]');
    if (!cards || !table || !toolbar) return;

    const key = 'bufalo:view:' + (root.dataset.viewKey || location.pathname);
    let mode = 'cards';
    try {
      const saved = localStorage.getItem(key);
      if (saved === 'table' || saved === 'cards') mode = saved;
    } catch (_) {}

    const group = document.createElement('div');
    group.className = 'entity-view-toggle';
    group.setAttribute('role', 'group');
    group.setAttribute('aria-label', 'Estilo de presentación');
    [['cards', 'Tarjetas', 'grid-3x2-gap'], ['table', 'Tabla', 'table']].forEach(function (item) {
      const button = document.createElement('button');
      button.type = 'button';
      button.className = 'btn btn-sm btn-outline-secondary';
      button.dataset.mode = item[0];
      const icon = document.createElement('i');
      icon.className = 'bi bi-' + item[2];
      icon.setAttribute('aria-hidden', 'true');
      const label = document.createElement('span');
      label.textContent = item[1];
      button.append(icon, label);
      button.addEventListener('click', function () { setMode(item[0], true); });
      group.appendChild(button);
    });
    toolbar.appendChild(group);

    function setMode(next, persist) {
      const isCards = next === 'cards';
      cards.hidden = !isCards;
      table.hidden = isCards;
      group.querySelectorAll('button').forEach(function (button) {
        const active = button.dataset.mode === next;
        button.setAttribute('aria-pressed', String(active));
        button.classList.toggle('btn-secondary', active);
        button.classList.toggle('btn-outline-secondary', !active);
      });
      root.dataset.mode = next;
      if (persist) {
        try { localStorage.setItem(key, next); } catch (_) {}
      }
    }

    setMode(mode, false);
  });
})();
