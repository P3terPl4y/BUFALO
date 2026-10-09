(function () {
    'use strict';
    if (!document.getElementById('healthPage')) return;
    const node = (id) => document.getElementById(id);
    const text = (id, value) => { node(id).textContent = value; };
    const number = (value, digits = 0) => Number(value || 0).toLocaleString('es', { maximumFractionDigits: digits });
    let history = { requests: [], errors: [], trend: [] };
    let tab = 'requests', page = 1, paused = false, busy = false;
    const pageSize = 25;
    const result = (status) => ({ 400: 'Solicitud inválida', 401: 'Sin autenticación', 403: 'Acceso rechazado', 404: 'No encontrado', 409: 'Conflicto', 413: 'Tamaño excedido', 422: 'Validación', 429: 'Límite de intentos', 500: 'Error interno', 502: 'Servicio inaccesible', 503: 'Servicio no disponible', 504: 'Tiempo agotado' }[status] || (status >= 500 ? 'Error del servidor' : status >= 400 ? 'Solicitud rechazada' : status >= 300 ? 'Redirección' : 'Correcta'));
    function renderRows() {
        const status = node('healthStatusFilter').value;
        const method = node('healthMethodFilter').value;
        const route = node('healthRouteFilter').value.trim().toLowerCase();
        const rows = (history[tab] || []).filter((row) => (status === 'all' || Math.floor(row.status / 100) === Number(status)) && (method === 'all' || row.method === method) && row.route.toLowerCase().includes(route));
        const pages = Math.max(1, Math.ceil(rows.length / pageSize));
        page = Math.min(page, pages);
        const body = node('healthHistoryRows');
        body.replaceChildren();
        for (const row of rows.slice((page - 1) * pageSize, page * pageSize)) {
            const tr = document.createElement('tr');
            const values = [new Date(row.at).toLocaleTimeString('es', { hour12: false }), row.method, row.route, String(row.status), `${number(row.duration_ms, 2)} ms`, result(row.status)];
            values.forEach((value, i) => {
                const td = document.createElement('td');
                td.textContent = value;
                if (i === 2) { td.className = 'bf-health__route'; td.title = value; }
                if (i === 3) { const badge = document.createElement('span'); badge.className = `bf-health__code bf-health__code--${Math.floor(row.status / 100)}`; badge.textContent = value; td.replaceChildren(badge); }
                tr.append(td);
            });
            body.append(tr);
        }
        if (!rows.length) {
            const tr = document.createElement('tr'), td = document.createElement('td');
            td.colSpan = 6; td.className = 'bf-health__empty';
            td.textContent = tab === 'errors' && !(history.errors || []).length ? 'No se han registrado errores en esta ventana.' : 'No hay peticiones que coincidan con estos filtros.';
            tr.append(td); body.append(tr);
        }
        text('healthVisibleCount', `${number(rows.length)} registros`);
        text('healthPageInfo', `Página ${page} de ${pages}`);
        node('healthPrev').disabled = page <= 1;
        node('healthNext').disabled = page >= pages;
        const params = new URLSearchParams({ format: 'csv', tab, status, method, route: node('healthRouteFilter').value.trim() });
        const csvExport = node('healthExportCSV'), excelExport = node('healthExportExcel');
        if (!csvExport || !excelExport) return;
        csvExport.href = `/admin/health/export?${params}`;
        params.set('format', 'excel');
        excelExport.href = `/admin/health/export?${params}`;
    }
    function renderTrend(points) {
        const chart = node('healthTrend');
        chart.replaceChildren();
        const maximum = Math.max(1, ...points.map((p) => Number(p.requests)));
        let requests = 0, errors = 0;
        for (const p of points) {
            const count = Number(p.requests || 0), failed = Number(p.client_errors || 0) + Number(p.server_errors || 0);
            requests += count; errors += failed;
            const bar = document.createElement('div'), error = document.createElement('span');
            bar.className = 'bf-health__bar';
            bar.style.height = `${Math.max(2, count / maximum * 100)}%`;
            error.style.height = `${count ? failed / count * 100 : 0}%`;
            bar.title = `${new Date(p.at).toLocaleTimeString('es', { hour: '2-digit', minute: '2-digit' })}: ${count} peticiones, ${failed} errores`;
            bar.append(error); chart.append(bar);
        }
        const summary = `${number(requests)} peticiones y ${number(errors)} errores durante la última hora.`;
        text('healthTrendSummary', summary); chart.setAttribute('aria-label', summary);
    }
    async function refresh() {
        if (busy) return;
        busy = true; node('healthRefresh').disabled = true;
        try {
            const response = await fetch('/admin/health/data', { credentials: 'same-origin', cache: 'no-store', redirect: 'error', headers: { Accept: 'application/json' }, signal: AbortSignal.timeout(8000) });
            if (!response.ok) throw new Error('health unavailable');
            const data = await response.json(), r = data.runtime, checks = data.checks || {};
            window.BufaloTraffic?.render(data.traffic_protection);
const outbox = data.notifications || {};
text("healthOutbox", outbox.status === "unavailable" ? "Cola no disponible" : `${number(outbox.pending)} pendientes · ${number(outbox.retrying)} con reintentos · antigüedad máxima ${number(outbox.oldest_seconds)} s`);
            const ok = data.status === 'ok';
            node('healthBanner').className = `bf-health__banner ${ok ? 'is-ok' : 'is-degraded'}`;
            text('healthStatus', ok ? 'Todos los servicios están operativos' : 'El sistema requiere atención');
            text('healthUpdated', `Última comprobación: ${new Date(data.checked_at).toLocaleString('es')} · ${paused ? 'actualización pausada' : 'cada 15 segundos'}`);
            const seconds = Number(r.uptime_seconds || 0);
            text('healthUptime', `Activo ${Math.floor(seconds / 86400)} d ${Math.floor(seconds % 86400 / 3600)} h ${Math.floor(seconds % 3600 / 60)} min`);
            for (const [name, id, description] of [['postgres', 'healthDB', 'SELECT 1'], ['redis', 'healthRedis', 'PING']]) {
                const check = checks[name] || {};
                text(id, check.status === 'available' ? 'Disponible' : 'No disponible');
                node(id).className = check.status === 'available' ? 'is-ok' : 'is-degraded';
                text(`${id}Latency`, `${description} · ${number(check.latency_ms, 1)} ms`);
            }
            text('healthRequests', number(r.requests)); text('healthRate', `${number(r.requests_per_minute, 2)} por minuto · promedio desde inicio`);
            text('healthLatency', `${number(r.average_latency_ms, 2)} ms`); text('healthP95', `p95 aproximado: ${r.p95_latency}`);
            text('healthErrors', number(r.server_errors_5xx)); text('healthErrorRate', `${number(r.error_rate_percent, 2)} % de 5xx · ${number(r.client_errors_4xx)} respuestas 4xx`);
            text('healthMemory', `${number(r.resident_memory_mb ?? r.heap_in_use_mb, 1)} MB ${r.resident_memory_mb == null ? 'heap' : 'RSS'}`); text('healthWorkers', `Heap: ${number(r.heap_in_use_mb, 1)} MB · ${number(r.goroutines)} goroutines · ${number(r.gc_cycles)} ciclos GC`);
            history = data.history || { requests: [], errors: [], trend: [] };
            text('healthRetention', `Últimas ${history.request_limit || 500} peticiones y ${history.error_limit || 200} errores · desde ${new Date(history.started_at).toLocaleString('es')}`);
            renderTrend(history.trend || []); renderRows();
        } catch (_) {
            node('healthBanner').className = 'bf-health__banner is-degraded';
            text('healthStatus', 'No se pudo actualizar la salud del sistema');
            text('healthUpdated', 'Los datos anteriores pueden estar desactualizados. Comprueba la conexión y tu sesión de administrador.');
        } finally { busy = false; node('healthRefresh').disabled = false; }
    }
    node('healthRefresh').addEventListener('click', refresh);
    node('healthPause').addEventListener('click', () => {
        paused = !paused; node('healthPause').textContent = paused ? 'Reanudar' : 'Pausar'; node('healthPause').setAttribute('aria-pressed', String(paused));
        if (!paused) refresh(); else text('healthUpdated', 'Actualización pausada. Puedes actualizar manualmente.');
    });
    document.querySelectorAll('[data-history-tab]').forEach((button) => button.addEventListener('click', () => {
        tab = button.dataset.historyTab; page = 1;
        document.querySelectorAll('[data-history-tab]').forEach((b) => { const active = b === button; b.classList.toggle('is-active', active); b.setAttribute('aria-pressed', String(active)); });
        renderRows();
    }));
    for (const id of ['healthStatusFilter', 'healthMethodFilter', 'healthRouteFilter']) node(id).addEventListener('input', () => { page = 1; renderRows(); });
    node('healthPrev').addEventListener('click', () => { page--; renderRows(); });
    node('healthNext').addEventListener('click', () => { page++; renderRows(); });
    node('healthExportPDF')?.addEventListener('click', () => window.print());
    document.addEventListener('visibilitychange', () => { if (!document.hidden && !paused) refresh(); });
    refresh(); window.setInterval(() => { if (!paused && !document.hidden) refresh(); }, 15000);
})();
