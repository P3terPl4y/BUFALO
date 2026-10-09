const { firefox } = require(process.env.PLAYWRIGHT_MODULE || 'playwright');
const assert = require('assert');
const fs = require('fs');
const path = require('path');

(async () => {
  const browser = await firefox.launch({ headless: true });
  const page = await browser.newPage({ viewport: { width: 390, height: 844 } });
  const errors = [];
  page.on('pageerror', error => errors.push(error.message));
  try {
    const script = path.resolve(__dirname, '../../public/js/entity-view-toggle.js');
    const html = `<div data-entity-view data-view-key="drivers">
      <header><h1>Choferes</h1><div data-view-toolbar></div></header>
      <section data-view-cards><article class="entity-card">Ana Driver</article></section>
      <section data-view-table hidden><table><tbody><tr><td>Ana Driver</td></tr></tbody></table></section>
    </div>
    <script>window.__viewPrefs = new Map(); Object.defineProperty(window, 'localStorage', { configurable: true, value: { getItem: key => window.__viewPrefs.get(key) || null, setItem: (key, value) => window.__viewPrefs.set(key, value) } });</script>`;
    await page.setContent(html);
    await page.addScriptTag({ path: script });
    assert.equal(await page.locator('[data-view-cards]').isVisible(), true, 'cards should be the default');
    assert.equal(await page.locator('[data-view-table]').isVisible(), false, 'table should start hidden');
    await page.getByRole('button', { name: 'Tabla' }).click();
    assert.equal(await page.locator('[data-view-cards]').isVisible(), false, 'table mode should hide cards');
    assert.equal(await page.locator('[data-view-table]').isVisible(), true, 'table mode should show the table');
    assert.equal(await page.getByRole('button', { name: 'Tabla' }).getAttribute('aria-pressed'), 'true');
    assert.equal(await page.evaluate(() => window.__viewPrefs.get('bufalo:view:drivers')), 'table', 'view choice should persist');
    await page.getByRole('button', { name: 'Tarjetas' }).click();
    assert.equal(await page.locator('[data-view-cards]').isVisible(), true, 'the toolbar should remain available to switch back');
    assert.equal(await page.locator('[data-view-table]').isVisible(), false);
    await page.evaluate(() => {
      window.__viewPrefs.set('bufalo:view:drivers', 'table');
      document.body.innerHTML = '<div data-entity-view data-view-key="drivers"><header><div data-view-toolbar></div></header><section data-view-cards>Tarjetas</section><section data-view-table hidden>Tabla</section></div>';
    });
    await page.addScriptTag({ path: script });
    assert.equal(await page.locator('[data-view-cards]').isVisible(), false, 'a saved table preference should be restored on a new page render');
    assert.equal(await page.locator('[data-view-table]').isVisible(), true);
    assert.deepEqual(errors, [], 'the toggle should not produce browser errors');
    console.log('PASS entity cards: default cards, table toggle, restored preference, return toggle');
  } finally {
    await browser.close();
  }
})().catch(error => { console.error(error); process.exitCode = 1; });
