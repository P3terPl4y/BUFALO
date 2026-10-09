(function () {
    'use strict';
    const root = document.getElementById('invoiceStudio');
    if (!root) return;
    const storageKey = 'bufalo.invoice-studio.v1';
    const palette = document.getElementById('studioPalette');
    const canvas = document.getElementById('studioCanvas');
    const paper = document.getElementById('studioPaper');
    const notice = document.getElementById('studioNotice');
    const modules = new Map();
    const maxModules = 16;
    const maxBlocks = 16;
    const maxSavedDesigns = 12;
    let initialized = false;
    const presets = {
        classic: { preset: 'classic', name: 'Factura clásica', format: 'a4', color: '#253746', currency: 'CUP', blocks: ['brand', 'parties', 'details', 'items', 'totals', 'payment', 'notes', 'signature'] },
        modern: { preset: 'modern', name: 'Factura moderna', format: 'a4', color: '#9a6810', currency: 'USD', blocks: ['brand', 'details', 'parties', 'items', 'totals', 'payment', 'signature'] },
        receipt: { preset: 'receipt', name: 'Recibo compacto', format: 'receipt', color: '#253746', currency: 'CUP', blocks: ['brand', 'details', 'items', 'totals', 'payment', 'notes'] }
    };
    // A block owns only its renderer. New invoice modules can register or remove
    // a block here without coupling it to ordering, persistence or print code.
    const definitions = [
        ['brand', 'Emisor y marca', 'Empresa, logo y contacto', 'building', () => `<div class="bf-paper-brand"><div><h2>Transportes del Caribe S.A.</h2><p>RUT 123456789 · La Habana, Cuba</p></div><span>FACTURA</span></div>`],
        ['parties', 'Participantes', 'Emisor y receptor', 'people', () => `<div class="bf-paper-grid"><section><small>EMISOR</small><strong>Transportes del Caribe S.A.</strong><span>facturas@bufalo.test</span></section><section><small>RECEPTOR</small><strong>Comercial La Estrella</strong><span>cliente@ejemplo.test</span></section></div>`],
        ['details', 'Datos de factura', 'Número, carga y fechas', 'card-list', () => `<div class="bf-paper-details"><span><small>NÚMERO</small><strong>BUF-2026-0042</strong></span><span><small>EMISIÓN</small><strong>09 oct 2026</strong></span><span><small>VENCIMIENTO</small><strong>23 oct 2026</strong></span><span><small>CARGA</small><strong>#1842</strong></span></div>`],
        ['items', 'Conceptos', 'Detalle y cantidad', 'list-check', (state) => `<table class="bf-paper-items"><thead><tr><th>Descripción</th><th>Cant.</th><th>Precio</th><th>Importe</th></tr></thead><tbody><tr><td>Transporte de mercancía · 240 km</td><td>1</td><td>125.00 ${safeText(state.currency)}</td><td>125.00 ${safeText(state.currency)}</td></tr><tr><td>Manipulación y entrega</td><td>1</td><td>20.00 ${safeText(state.currency)}</td><td>20.00 ${safeText(state.currency)}</td></tr></tbody></table>`],
        ['totals', 'Subtotal e impuestos', 'Moneda y total', 'calculator', (state) => `<div class="bf-paper-totals"><span>Subtotal <b>145.00 ${safeText(state.currency)}</b></span><span>Impuestos <b>14.50 ${safeText(state.currency)}</b></span><strong>Total <b>159.50 ${safeText(state.currency)}</b></strong></div>`],
        ['payment', 'Pago', 'Estado y método', 'credit-card', () => `<div class="bf-paper-payment"><strong>Estado: Pendiente</strong><span>Método previsto: Transferencia bancaria</span></div>`],
        ['notes', 'Notas', 'Condiciones y mensaje', 'chat-left-text', () => `<div class="bf-paper-notes"><small>NOTAS Y CONDICIONES</small><p>Gracias por confiar en nosotros. Pago a 14 días a partir de la fecha de emisión.</p></div>`],
        ['signature', 'Firma y cierre', 'Firma y agradecimiento', 'pen', () => `<div class="bf-paper-signature"><span>Firma autorizada</span><strong>Gracias por su preferencia</strong></div>`]
    ];
    function attachModule(id, module) {
        if (!/^[a-z][a-z0-9_-]{0,31}$/.test(id) || modules.has(id) || modules.size >= maxModules || !module || typeof module.label !== 'string' || typeof module.description !== 'string' || typeof module.render !== 'function') return false;
        modules.set(id, module);
        const button = document.createElement('button'); button.type = 'button'; button.draggable = true; button.dataset.block = id;
        const label = document.createElement('span'); label.innerHTML = `<i class="bi bi-${typeof module.icon === 'string' && /^[a-z0-9-]+$/.test(module.icon) ? module.icon : 'puzzle'}"></i> ${safeText(module.label.slice(0, 80))}`;
        const description = document.createElement('small'); description.textContent = module.description.slice(0, 80);
        button.append(label, description); wirePaletteButton(button); palette.append(button);
        if (initialized) render();
        return true;
    }
    function detachModule(id) {
        if (!modules.delete(id)) return false;
        state.blocks = state.blocks.filter((block) => block !== id);
        [...palette.querySelectorAll('[data-block]')].find((button) => button.dataset.block === id)?.remove(); render(); return true;
    }
    definitions.forEach(([id, label, description, icon, renderBlock]) => attachModule(id, { label, description, icon, render: renderBlock }));
    let state = loadState();
    initialized = true;
    function loadState() {
        try {
            const stored = localStorage.getItem(storageKey) || 'null';
            if (stored.length > 4096) return { ...presets.classic };
            const saved = JSON.parse(stored);
            if (saved) return sanitizeState(saved);
        } catch (_) { /* Ignore malformed or unavailable browser storage. */ }
        return { ...presets.classic };
    }
    function savedDesigns() {
        try {
            const stored = localStorage.getItem(`${storageKey}.saved`) || '[]';
            if (stored.length > 65536) return [];
            const designs = JSON.parse(stored);
            if (!Array.isArray(designs)) return [];
            return designs.slice(0, maxSavedDesigns)
                .filter((item) => item && typeof item === 'object' && typeof item.name === 'string' && Array.isArray(item.blocks) && item.blocks.length <= maxBlocks && new Set(item.blocks).size === item.blocks.length && item.blocks.every((block) => typeof block === 'string' && /^[a-z][a-z0-9_-]{0,31}$/.test(block)))
                .map((item) => ({ ...sanitizeState(item), name: item.name.slice(0, 48) }));
        } catch (_) { return []; }
    }
    function safeText(value) { return String(value).replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c])); }
    function sanitizeState(value) {
        if (!value || !Array.isArray(value.blocks) || value.blocks.length > maxBlocks || new Set(value.blocks).size !== value.blocks.length || !value.blocks.every((block) => typeof block === 'string' && /^[a-z][a-z0-9_-]{0,31}$/.test(block))) return { ...presets.classic };
        return { name: typeof value.name === 'string' ? value.name.slice(0, 48) : presets.classic.name,
            format: ['a4', 'letter', 'receipt'].includes(value.format) ? value.format : 'a4',
            color: typeof value.color === 'string' && /^#[0-9a-f]{6}$/i.test(value.color) ? value.color : '#253746',
            currency: ['CUP', 'USD', 'EUR', 'MLC'].includes(value.currency) ? value.currency : 'CUP',
            blocks: [...value.blocks], preset: ['classic', 'modern', 'receipt'].includes(value.preset) ? value.preset : '' };
    }
    function render() {
        paper.dataset.format = state.format;
        paper.style.setProperty('--invoice-accent', /^#[0-9a-f]{6}$/i.test(state.color) ? state.color : '#253746');
        document.getElementById('studioName').value = state.name;
        document.getElementById('studioFormat').value = state.format;
        document.getElementById('studioColor').value = state.color;
        document.getElementById('studioCurrency').value = state.currency;
        document.getElementById('studioFormatLabel').textContent = ({ a4: 'A4 · vertical', letter: 'Carta · vertical', receipt: 'Recibo · 80 mm' })[state.format];
        canvas.replaceChildren();
        for (const [index, type] of state.blocks.entries()) {
            const block = document.createElement('section');
            block.className = 'bf-paper-block'; block.dataset.index = String(index); block.dataset.type = type; block.draggable = true;
            const heading = document.createElement('div'); heading.className = 'bf-paper-block__toolbar';
            const definition = modules.get(type);
            const label = definition ? definition.label : `Módulo pendiente: ${type}`;
            const title = document.createElement('strong'); title.textContent = label;
            const actions = document.createElement('span');
            for (const [label, movement] of [['↑', -1], ['↓', 1], ['Quitar', 0]]) {
                const button = document.createElement('button'); button.type = 'button'; button.textContent = label;
                button.setAttribute('aria-label', movement === 0 ? `Quitar ${label}` : `${movement < 0 ? 'Subir' : 'Bajar'} ${label}`);
                button.disabled = movement < 0 && index === 0 || movement > 0 && index === state.blocks.length - 1;
                button.addEventListener('click', () => { if (!movement) state.blocks.splice(index, 1); else [state.blocks[index], state.blocks[index + movement]] = [state.blocks[index + movement], state.blocks[index]]; render(); });
                actions.append(button);
            }
            actions.append(title); heading.append(actions);
            const content = document.createElement('div'); content.className = `bf-paper-content bf-paper-content--${type}`;
            try {
                if (!definition) content.textContent = 'Este bloque aún no está acoplado en esta sesión.';
                else {
                    const output = definition.render(state);
                    if (typeof output === 'string') content.innerHTML = output;
                    else content.textContent = 'Este bloque no devolvió contenido válido.';
                }
            } catch (_) { content.textContent = 'No se pudo mostrar este bloque.'; }
            block.append(heading, content); canvas.append(block);
        }
        palette.querySelectorAll('[data-block]').forEach((button) => { button.disabled = state.blocks.includes(button.dataset.block); button.classList.toggle('is-added', button.disabled); });
        document.querySelectorAll('[data-preset]').forEach((button) => button.classList.toggle('is-active', button.dataset.preset === state.preset));
        renderSaved();
    }
    function renderSaved() {
        const select = document.getElementById('studioSaved'), current = select.value;
        const designs = savedDesigns();
        select.replaceChildren(new Option('Seleccionar diseño guardado', ''));
        designs.forEach((design, index) => select.add(new Option(design.name, String(index))));
        if ([...select.options].some((option) => option.value === current)) select.value = current;
    }
    function save() {
        state.name = document.getElementById('studioName').value.trim().slice(0, 48) || 'Diseño sin nombre';
        try {
            const designs = savedDesigns();
            const next = [{ ...state, blocks: [...state.blocks] }, ...designs.filter((item) => item.name !== state.name)].slice(0, maxSavedDesigns);
            localStorage.setItem(`${storageKey}.saved`, JSON.stringify(next));
            notice.textContent = `Diseño «${state.name}» guardado en este navegador.`; renderSaved();
        } catch (_) { notice.textContent = 'El navegador no permitió guardar el diseño. Puedes seguir usando la vista previa.'; }
    }
    function wirePaletteButton(button) {
        button.addEventListener('dragstart', (event) => { event.dataTransfer.setData('text/bufalo-invoice-module', button.dataset.block); event.dataTransfer.effectAllowed = 'copy'; });
        button.addEventListener('click', () => { if (modules.has(button.dataset.block) && !state.blocks.includes(button.dataset.block)) { state.blocks.push(button.dataset.block); render(); } });
    }
    canvas.addEventListener('dragstart', (event) => { const block = event.target.closest('.bf-paper-block'); if (block) { event.dataTransfer.setData('text/bufalo-invoice-order', block.dataset.index); event.dataTransfer.effectAllowed = 'move'; } });
    canvas.addEventListener('dragover', (event) => { if (event.target.closest('.bf-paper-block') || event.target === canvas) { event.preventDefault(); event.dataTransfer.dropEffect = event.dataTransfer.types.includes('text/bufalo-invoice-order') ? 'move' : 'copy'; } });
    canvas.addEventListener('drop', (event) => {
        const target = event.target.closest('.bf-paper-block');
        const targetIndex = target ? Number(target.dataset.index) : state.blocks.length;
        const newBlock = event.dataTransfer.getData('text/bufalo-invoice-module');
        if (newBlock) {
            if (modules.has(newBlock) && !state.blocks.includes(newBlock) && state.blocks.length < maxBlocks) state.blocks.splice(Math.min(targetIndex, state.blocks.length), 0, newBlock);
        } else if (event.dataTransfer.types.includes('text/bufalo-invoice-order')) {
            const source = Number(event.dataTransfer.getData('text/bufalo-invoice-order'));
            if (Number.isInteger(source) && source >= 0 && source < state.blocks.length) { const [item] = state.blocks.splice(source, 1); const adjustedTarget = targetIndex > source ? targetIndex - 1 : targetIndex; state.blocks.splice(Math.min(adjustedTarget, state.blocks.length), 0, item); }
        }
        event.preventDefault(); render();
    });
    document.querySelectorAll('[data-preset]').forEach((button) => button.addEventListener('click', () => { state = { ...presets[button.dataset.preset], preset: button.dataset.preset }; render(); }));
    document.getElementById('studioFormat').addEventListener('change', (event) => { state.format = ['a4', 'letter', 'receipt'].includes(event.target.value) ? event.target.value : 'a4'; render(); });
    document.getElementById('studioColor').addEventListener('input', (event) => { state.color = event.target.value; render(); });
    document.getElementById('studioCurrency').addEventListener('change', (event) => { state.currency = ['CUP', 'USD', 'EUR', 'MLC'].includes(event.target.value) ? event.target.value : 'CUP'; render(); });
    document.getElementById('studioSave').addEventListener('click', save);
    document.getElementById('studioReset').addEventListener('click', () => { state = { ...presets.classic }; notice.textContent = 'Se restauró el formato clásico. Tus diseños guardados siguen disponibles.'; render(); });
    document.getElementById('studioClear').addEventListener('click', () => { state.blocks = []; render(); });
    document.getElementById('studioPrint').addEventListener('click', () => window.print());
    document.getElementById('studioSaved').addEventListener('change', (event) => { if (event.target.value === '') return; try { const saved = JSON.parse(localStorage.getItem(`${storageKey}.saved`) || '[]')[Number(event.target.value)]; if (saved) state = sanitizeState(saved); render(); } catch (_) { notice.textContent = 'No se pudo abrir el diseño guardado.'; } });
    // Stable extension seam: feature bundles can attach or detach invoice blocks
    // without changing the canvas, drag behavior, persistence or print surface.
    window.BufaloInvoiceStudio = Object.freeze({ attachModule, detachModule });
    render();
})();
