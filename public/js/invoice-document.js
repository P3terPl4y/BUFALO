(function () {
    'use strict';
    const root = document.getElementById('invoicePresentation');
    if (!root) return;
    let data;
    try { data = JSON.parse(root.dataset.invoice || '{}'); } catch (_) { return; }
    const invoice = data.invoice || {}, template = data.template || {};
    const allowed = new Set(['brand', 'parties', 'details', 'items', 'totals', 'payment', 'notes', 'signature']);
    const blocks = Array.isArray(template.blocks) ? template.blocks.filter((id, index, all) => allowed.has(id) && all.indexOf(id) === index).slice(0, 16) : [];
    root.dataset.format = ['a4', 'letter', 'receipt'].includes(template.format) ? template.format : 'a4';
    if (/^#[0-9a-f]{6}$/i.test(template.color || '')) root.style.setProperty('--invoice-accent', template.color);

    const node = (tag, className, content) => {
        const element = document.createElement(tag);
        if (className) element.className = className;
        if (content !== undefined) element.textContent = String(content);
        return element;
    };
    const text = (value) => value === undefined || value === null || value === '' ? '—' : String(value);
    const currency = text(invoice.currency);
    const money = (value) => `${text(value)} ${currency}`;
    const renderers = {
        brand() {
            const box = node('div', 'bf-paper-brand');
            const identity = node('div');
            identity.append(node('h2', '', text(invoice.issuer)));
            identity.append(node('p', '', text(invoice.load_reference)));
            box.append(identity, node('span', '', 'FACTURA'));
            return box;
        },
        parties() {
            const box = node('div', 'bf-paper-grid');
            for (const [label, value] of [['EMISOR', invoice.issuer], ['RECEPTOR', invoice.recipient]]) {
                const section = node('section');
                section.append(node('small', '', label), node('strong', '', text(value)));
                box.append(section);
            }
            return box;
        },
        details() {
            const box = node('div', 'bf-paper-details');
            for (const [label, value] of [['NÚMERO', invoice.number], ['EMISIÓN', invoice.issue_date], ['VENCIMIENTO', invoice.due_date], ['CARGA', invoice.load_reference]]) {
                const item = node('span');
                item.append(node('small', '', label), node('strong', '', text(value)));
                box.append(item);
            }
            return box;
        },
        items() {
            const table = node('table', 'bf-paper-items');
            const head = node('thead'), heading = node('tr');
            for (const title of ['Descripción', 'Distancia', 'Tarifa', 'Importe']) heading.append(node('th', '', title));
            head.append(heading);
            const body = node('tbody'), row = node('tr');
            for (const value of [`Transporte · ${text(invoice.load_reference)}`, `${text(invoice.distance_km)} km`, money(invoice.rate_per_km), money(invoice.subtotal)]) row.append(node('td', '', value));
            body.append(row); table.append(head, body);
            return table;
        },
        totals() {
            const box = node('div', 'bf-paper-totals');
            for (const [label, value] of [['Subtotal', invoice.subtotal], ['Impuestos', invoice.taxes]]) {
                const row = node('span'); row.append(node('span', '', label), node('b', '', money(value))); box.append(row);
            }
            const total = node('strong'); total.append(node('span', '', 'Total'), node('b', '', money(invoice.total))); box.append(total);
            return box;
        },
        payment() {
            const box = node('div', 'bf-paper-payment');
            box.append(node('strong', '', `Estado: ${text(invoice.status)}`), node('span', '', `Método: ${text(invoice.payment_method)}`));
            return box;
        },
        notes() {
            const box = node('div', 'bf-paper-notes');
            box.append(node('small', '', 'NOTAS Y CONDICIONES'), node('p', '', 'Documento asociado a la carga y a las partes indicadas.'));
            return box;
        },
        signature() {
            const box = node('div', 'bf-paper-signature');
            box.append(node('span', '', 'Firma autorizada'), node('strong', '', 'Documento emitido por BUFALO'));
            return box;
        }
    };
    const top = node('header', 'bf-invoice-paper__top');
    top.append(node('strong', '', 'BUFALO'), node('span', '', text(template.name)));
    const canvas = node('div', 'bf-invoice-paper__canvas');
    for (const id of blocks) {
        const section = node('section', 'bf-paper-block');
        section.dataset.type = id;
        section.append(renderers[id]());
        canvas.append(section);
    }
    const footer = node('footer', 'bf-invoice-paper__footer', `Factura ${text(invoice.number)} · ${money(invoice.total)}`);
    root.replaceChildren(top, canvas, footer);
    document.getElementById('invoicePresentationPrint')?.addEventListener('click', () => window.print());
})();
