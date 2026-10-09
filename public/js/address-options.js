(() => {
  'use strict';
  const selectors = [...document.querySelectorAll('select[name="origen_direccion_id"], select[name="destino_direccion_id"], select[name="direccion_id"]')];
  if (!selectors.length) return;
  const button = document.createElement('button');
  button.type = 'button'; button.className = 'btn btn-secondary';
  button.textContent = 'Cargar más direcciones';
  selectors[0].parentElement.append(button);
  let page = 2;
  button.addEventListener('click', async () => {
    button.disabled = true;
    try {
      const response = await fetch('/direcciones/options?page=' + page, {credentials: 'same-origin'});
      if (!response.ok) throw new Error('No se pudieron cargar las direcciones');
      const {data} = await response.json();
      selectors.forEach(select => {
        const existing = new Set([...select.options].map(option => option.value));
        data.forEach(address => {
          if (existing.has(String(address.id))) return;
          const option = new Option(address.ciudad + ' — ' + address.estado_provincia + ' (' + address.pais + ')', String(address.id));
          option.dataset.city = address.ciudad; option.dataset.state = address.estado_provincia;
          select.add(option);
        });
      });
      page++;
      button.textContent = data.length < 100 ? 'Todas las direcciones cargadas' : 'Cargar más direcciones';
      button.disabled = data.length < 100;
    } catch (_) {
      button.textContent = 'Error al cargar. Reintentar'; button.disabled = false;
    }
  });
})();
