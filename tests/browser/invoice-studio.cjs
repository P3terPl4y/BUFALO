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
    const view = fs.readFileSync(viewPath, 'utf8').replace('<script src="/js/invoice-studio.js" defer></script>', '');
    const storageBootstrap = `<script>
      (() => {
        const entries = new Map([
          ['bufalo.invoice-studio.v1', JSON.stringify({ name: 'Diseño con módulo externo', format: 'letter', color: '#123456', currency: 'USD', blocks: ['addon_tax'], preset: '' })],
          ['bufalo.invoice-studio.v1.saved', JSON.stringify([{ name: 'Diseño con módulo externo', format: 'receipt', color: '#654321', currency: 'EUR', blocks: ['addon_tax'], preset: '' }])]
        ]);
        Object.defineProperty(window, 'localStorage', { configurable: true, value: {
          getItem: key => entries.has(key) ? entries.get(key) : null,
          setItem: (key, value) => entries.set(key, String(value)), removeItem: key => entries.delete(key)
        } });
        window.print = () => { window.__printCalled = true; };
      })();
    </script>`;
    await page.setContent(storageBootstrap + view);
    await page.addScriptTag({ path: scriptPath });

    assert.equal(await page.locator('#studioPalette [data-block]').count(), 8, 'each built-in block should appear once');
    assert.equal(await page.locator('#studioCanvas [data-type]').count(), 1, 'unknown extension blocks should remain visible while detached');
    assert((await page.locator('#studioCanvas').innerText()).includes('aún no está acoplado'));
    assert.equal(await page.locator('#studioSaved option').count(), 2, 'saved designs with an unloaded module should be retained');

    const attached = await page.evaluate(() => window.BufaloInvoiceStudio.attachModule('addon_tax', {
      label: '<Impuesto>', description: 'Módulo conectado tras iniciar el editor', icon: 'receipt',
      render: () => '<strong>Impuesto cargado</strong>'
    }));
    assert.equal(attached, true);
    assert((await page.locator('#studioCanvas').innerText()).includes('Impuesto cargado'));
    assert.equal(await page.locator('#studioPalette [data-block]').count(), 9, 'attaching one extension should add exactly one palette module');

    await page.locator('#studioSaved').selectOption('0');
    assert.equal(await page.locator('#studioPaper').getAttribute('data-format'), 'receipt');
    assert((await page.locator('#studioPalette').innerText()).includes('<Impuesto>'));
    assert(await page.locator('#studioPalette [data-block="addon_tax"]').isDisabled(), 'existing blocks should be disabled in the palette');
    assert.equal(await page.locator('#studioCanvas [data-type]').count(), 1, 'a block cannot be added twice');

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
    assert((await page.locator('#studioNotice').innerText()).includes('guardado'));
    const saved = await page.evaluate(() => JSON.parse(localStorage.getItem('bufalo.invoice-studio.v1.saved')));
    assert.equal(saved[0].name, 'Mi factura de prueba');
    assert.equal(saved[0].blocks[0], 'notes');
    await page.locator('#studioSaved').selectOption('0');
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
    for (const id of ['brand', 'parties', 'details', 'items', 'totals', 'payment', 'notes', 'signature', 'addon_tax', ...Array.from({ length: 7 }, (_, index) => `extra_${index}`)]) {
      await page.locator(`#studioPalette [data-block="${id}"]`).click();
    }
    assert.equal(await page.locator('#studioCanvas [data-type]').count(), 16, 'the block limit should match the module limit');

    await page.setViewportSize({ width: 390, height: 844 });
    assert(await page.locator('#studioPalette').isVisible(), 'mobile palette should remain visible');
    await page.locator('#studioPrint').click();
    assert.equal(await page.evaluate(() => window.__printCalled), true);
    assert.deepEqual(errors, [], `browser errors: ${errors.join('; ')}`);
    console.log('PASS invoice designer: registry, deferred modules, drag ordering, persistence, responsive view, print');
  } finally {
    await browser.close();
  }
})().catch(error => { console.error(error); process.exit(1); });
