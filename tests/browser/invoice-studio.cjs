const { firefox } = require(process.env.PLAYWRIGHT_MODULE || 'playwright');
const assert = require('assert');
const fs = require('fs');
const path = require('path');

(async () => {
  const browser = await firefox.launch({ headless: true });
  const page = await browser.newPage({ viewport: { width: 1280, height: 900 } });
  const errors = [];
  page.on('pageerror', error => errors.push(error.message));
  try {
    const viewPath = path.resolve(__dirname, '../../app/views/admin/invoice_studio.html');
    const scriptPath = path.resolve(__dirname, '../../public/js/invoice-studio.js');
    const view = fs.readFileSync(viewPath, 'utf8')
      .replace('<script src="/js/invoice-studio.js" defer></script>', '')
      .replace('{{ .templatesEndpoint }}', '/facturas/plantillas')
      .replace('{{ .csrfToken }}', 'test-csrf');
    const storageBootstrap = `<script>
      (() => {
        let nextTemplateID = 78;
        window.__remoteTemplates = [{ id: 77, name: 'Diseño guardado por broker', preset: 'receipt', format: 'receipt', color: '#654321', currency: 'EUR', blocks: ['brand', 'parties', 'details', 'totals'], default: true }];
        const entries = new Map([
          ['bufalo.invoice-studio.v1', JSON.stringify({ name: 'Diseño con módulo externo', format: 'letter', color: '#123456', currency: 'USD', blocks: ['addon_tax'], preset: '' })],
        ]);
        Object.defineProperty(window, 'localStorage', { configurable: true, value: {
          getItem: key => entries.has(key) ? entries.get(key) : null,
          setItem: (key, value) => entries.set(key, String(value)), removeItem: key => entries.delete(key)
        } });
        window.fetch = async (_url, options = {}) => {
          if (!options.method || options.method === 'GET') return { ok: true, json: async () => ({ data: structuredClone(window.__remoteTemplates) }) };
          const form = new URLSearchParams(options.body || '');
          if (form.get('_csrf') !== 'test-csrf') return { ok: false, json: async () => ({}) };
          const config = JSON.parse(form.get('config'));
          if (!config.id) config.id = nextTemplateID++;
          const saved = { id: config.id, name: config.name, preset: config.preset, format: config.format, color: config.color, currency: config.currency, blocks: config.blocks, default: config.default };
          window.__remoteTemplates = [saved, ...window.__remoteTemplates.filter(item => item.id !== saved.id).map(item => saved.default ? { ...item, default: false } : item)];
          return { ok: true, status: 201, json: async () => ({ data: structuredClone(saved) }) };
        };
        window.print = () => { window.__printCalled = true; };
      })();
    </script>`;
    await page.setContent(storageBootstrap + view);
    await page.addScriptTag({ path: scriptPath });

    assert.equal(await page.locator('#studioPalette [data-block]').count(), 8, 'each built-in block should appear once');
    assert.equal(await page.locator('#studioCanvas [data-type]').count(), 1, 'unknown extension blocks should remain visible while detached');
    assert((await page.locator('#studioCanvas').innerText()).includes('aún no está acoplado'));
    await page.waitForFunction(() => document.querySelectorAll('#studioSaved option').length === 2);
    assert.equal(await page.locator('#studioSaved option').count(), 2, 'broker templates should be loaded from the server');

    const attached = await page.evaluate(() => window.BufaloInvoiceStudio.attachModule('addon_tax', {
      label: '<Impuesto>', description: 'Módulo conectado tras iniciar el editor', icon: 'receipt',
      render: () => '<strong>Impuesto cargado</strong>'
    }));
    assert.equal(attached, true);
    assert((await page.locator('#studioCanvas').innerText()).includes('Impuesto cargado'));
    assert.equal(await page.locator('#studioPalette [data-block]').count(), 9, 'attaching one extension should add exactly one palette module');

    await page.locator('#studioSaved').selectOption('77');
    assert.equal(await page.locator('#studioPaper').getAttribute('data-format'), 'receipt');
    assert((await page.locator('#studioPalette').innerText()).includes('<Impuesto>'));
    assert(await page.locator('#studioPalette [data-block="addon_tax"]').isEnabled(), 'the broker template should contain only registered persistent blocks');
    assert.equal(await page.locator('#studioCanvas [data-type]').count(), 4, 'the selected broker template should restore its ordered blocks');

    await page.locator('[data-preset="classic"]').click();
    await page.evaluate(() => {
      const transfer = new DataTransfer();
      const source = document.querySelector('[data-type="notes"]');
      const target = document.querySelector('[data-type="brand"]');
      source.dispatchEvent(new DragEvent('dragstart', { bubbles: true, dataTransfer: transfer }));
      target.dispatchEvent(new DragEvent('dragover', { bubbles: true, cancelable: true, dataTransfer: transfer }));
      target.dispatchEvent(new DragEvent('drop', { bubbles: true, cancelable: true, dataTransfer: transfer }));
    });
    assert.equal(await page.locator('#studioCanvas [data-type]').first().getAttribute('data-type'), 'notes');
    await page.locator('#studioName').fill('Mi factura de prueba');
    await page.locator('#studioSave').click();
    await page.waitForFunction(() => document.querySelector('#studioNotice').innerText.includes('guardado'));
    assert((await page.locator('#studioNotice').innerText()).includes('guardado'));
    const saved = await page.evaluate(() => window.__remoteTemplates[0]);
    assert.equal(saved.name, 'Mi factura de prueba');
    assert.equal(saved.blocks[0], 'notes');
    await page.locator('#studioSaved').selectOption(String(saved.id));
    assert.equal(await page.locator('#studioCanvas [data-type]').first().getAttribute('data-type'), 'notes');

    await page.locator('#studioClear').click();
    for (let index = 0; index < 7; index++) {
      const id = `extra_${index}`;
      assert.equal(await page.evaluate(id => window.BufaloInvoiceStudio.attachModule(id, {
        label: `Extensión ${id}`, description: 'Bloque adicional', render: () => '<span>Extensión</span>'
      }), id), true);
    }
    assert.equal(await page.evaluate(() => window.BufaloInvoiceStudio.attachModule('overflow', {
      label: 'Exceso', description: 'Debe rechazarse', render: () => ''
    })), false, 'the module registry should stay bounded');
    for (const id of ['brand', 'items', 'payment', 'notes', 'signature', 'addon_tax', ...Array.from({ length: 7 }, (_, index) => `extra_${index}`)]) {
      await page.locator(`#studioPalette [data-block="${id}"]`).click();
    }
    assert.equal(await page.locator('#studioCanvas [data-type]').count(), 16, 'the block limit should match the module limit');

    await page.setViewportSize({ width: 390, height: 844 });
    assert(await page.locator('#studioPalette').isVisible(), 'mobile palette should remain visible');
    await page.locator('#studioPrint').click();
    assert.equal(await page.evaluate(() => window.__printCalled), true);

    const documentScript = path.resolve(__dirname, '../../public/js/invoice-document.js');
    await page.setContent('<button id="invoicePresentationPrint">Imprimir</button><article class="bf-invoice-paper" id="invoicePresentation"></article>');
    await page.evaluate(() => {
      document.getElementById('invoicePresentation').dataset.invoice = JSON.stringify({
        template: { name: 'Formato del broker', format: 'letter', color: '#123456', blocks: ['parties', 'details', 'totals'] },
        invoice: { issuer: '<img src=x onerror=alert(1)>', recipient: 'Chofer QA', number: 'INV-QA-1', issue_date: '09/10/2026', due_date: 'Sin vencimiento', load_reference: 'CARGA-QA-9', distance_km: '8.00', rate_per_km: '2.0000', subtotal: '16.00', taxes: '0.00', total: '16.00', currency: 'USD', status: 'borrador', payment_method: 'Pendiente de registrar' }
      });
      window.__printCalled = false;
      window.print = () => { window.__printCalled = true; };
    });
    await page.addScriptTag({ path: documentScript });
    assert.equal(await page.locator('#invoicePresentation').getAttribute('data-format'), 'letter');
    assert.deepEqual(await page.locator('#invoicePresentation [data-type]').evaluateAll(nodes => nodes.map(node => node.dataset.type)), ['parties', 'details', 'totals']);
    assert((await page.locator('#invoicePresentation').innerText()).includes('<img src=x onerror=alert(1)>'), 'invoice values should render as text');
    assert.equal(await page.locator('#invoicePresentation img').count(), 0, 'invoice values must not create markup');
    await page.locator('#invoicePresentationPrint').click();
    assert.equal(await page.evaluate(() => window.__printCalled), true);
    assert.deepEqual(errors, [], `browser errors: ${errors.join('; ')}`);
    console.log('PASS invoices: persistent template editor, responsive drag ordering, real invoice rendering, escaped data, print');
  } finally {
    let closeTimer;
    await Promise.race([browser.close(), new Promise(resolve => { closeTimer = setTimeout(resolve, 3000); })]);
    clearTimeout(closeTimer);
  }
})().then(() => process.exit(0)).catch(error => { console.error(error); process.exit(1); });
