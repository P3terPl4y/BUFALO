(function () {
    'use strict';

    const text = (id, value) => {
        const node = document.getElementById(id);
        if (node) node.textContent = value;
    };
    const number = (value, digits) => Number(value || 0).toLocaleString('es', {
        maximumFractionDigits: digits === undefined ? 0 : digits,
    });
    const duration = (seconds) => {
        const value = Math.max(0, Math.floor(Number(seconds || 0)));
        const days = Math.floor(value / 86400);
        const hours = Math.floor((value % 86400) / 3600);
        const minutes = Math.floor((value % 3600) / 60);
        return `${days ? `${days} d ` : ''}${hours} h ${minutes} min`;
    };

    async function refreshHealth() {
        const status = document.getElementById('metricsHealthStatus');
        try {
            const response = await fetch('/admin/health', {
                credentials: 'same-origin', cache: 'no-store', headers: { Accept: 'application/json' },
            });
            if (!response.ok) throw new Error(`HTTP ${response.status}`);
            const data = await response.json();
            window.BufaloTraffic?.render(data.traffic_protection);
            const runtime = data.runtime || {};
            const dependencies = data.dependencies || {};
            text('metricsPostgres', dependencies.postgres_tcp?.status === 'available' ? 'Disponible' : 'No disponible');
            text('metricsPostgresLatency', `${number(dependencies.postgres_tcp?.latency_ms, 1)} ms TCP`);
            text('metricsRedis', dependencies.redis_tcp?.status === 'available' ? 'Disponible' : 'No disponible');
            text('metricsRedisLatency', `${number(dependencies.redis_tcp?.latency_ms, 1)} ms TCP`);
            text('metricsUptime', duration(runtime.uptime_seconds));
            text('metricsRequests', number(runtime.requests));
            text('metricsRequestRate', `${number(runtime.requests_per_minute, 2)} solicitudes/min promedio`);
            text('metricsLatency', `${number(runtime.average_latency_ms, 2)} ms`);
            text('metricsP95', `p95: ${runtime.p95_latency || '—'}`);
            text('metricsErrors', number(runtime.server_errors_5xx));
            text('metricsErrorRate', `${number(runtime.error_rate_percent, 2)} % de solicitudes`);
            text('metricsHeap', runtime.resident_memory_mb == null ? `${number(runtime.heap_in_use_mb, 1)} MB heap · RSS no disponible` : `${number(runtime.resident_memory_mb, 1)} MB RSS`);
            text('metricsHeapAlloc', `${number(runtime.heap_alloc_mb, 1)} MB asignados · ${number(runtime.total_alloc_mb, 1)} MB acumulados`);
            text('metricsGoroutines', number(runtime.goroutines));
            text('metricsGC', `GOMAXPROCS: ${number(runtime.gomaxprocs)} · GC: ${number(runtime.gc_cycles)}`);
            text('metricsHealthChecked', `Actualizado: ${data.checked_at || '—'}. p95 aproximado por intervalos.`);
            if (status) {
                status.textContent = data.status === 'ok' ? 'Saludable' : 'Degradado';
                status.className = `badge ${data.status === 'ok' ? 'bg-success' : 'bg-warning text-dark'}`;
            }
        } catch (_) {
            if (status) {
                status.textContent = 'No disponible';
                status.className = 'badge bg-danger';
            }
            text('metricsHealthChecked', 'No se pudo obtener la telemetría. Comprueba el acceso al panel de salud.');
        }
    }

    refreshHealth();
    window.setInterval(refreshHealth, 15000);
})();
