(function () {
    'use strict';
    const text = (id, value) => { const element = document.getElementById(id); if (element) element.textContent = value; };
    const number = (value) => Number(value || 0).toLocaleString('es');
    window.BufaloTraffic = {
        render(data) {
            if (!document.getElementById('trafficMode')) return;
            const p = data?.policy || {};
            const scan = data?.security_scan || {};
            const statuses = {passed:'Completado sin vulnerabilidades detectadas', failed:'Análisis fallido o con hallazgos; revisar el informe', running:'Análisis en curso', not_run:'Aún no se ha ejecutado', unavailable:'Informe no disponible'};
            text('securityScanStatus', statuses[scan.status] || 'Informe no disponible');
            const mb = value => (Number(value || 0) / 1048576).toFixed(1);
            text('securityScanResources', scan.started_at ? `Modo: ${scan.mode || 'sin especificar'} · pico del grupo: ${mb(scan.memory_peak_bytes)} MiB · límite: ${mb(scan.memory_limit_bytes)} MiB · CPU: ${Number(scan.cpu_seconds || 0).toFixed(1)} s · duración: ${Number(scan.duration_seconds || 0).toFixed(1)} s · resultado: ${scan.result || 'pendiente'} · inicio: ${new Date(scan.started_at).toLocaleString('es')}` : 'El consumo se registra al ejecutar el analizador aislado.');
            const body = document.getElementById('securityRejections');
            if (body) {
                body.replaceChildren();
                for (const event of (data?.recent_rejections || []).slice(0,128)) {
                    const row = document.createElement('tr');
                    for (const value of [new Date(event.at).toLocaleString('es'),event.outcome,event.status,event.identity,number(event.count)]) {
                        const cell = document.createElement('td'); cell.textContent = value; row.appendChild(cell);
                    }
                    body.appendChild(row);
                }
            }

            const mode = p.mode === 'enforce' ? 'Bloqueo activo' : p.mode === 'observe' ? 'Solo observación' : 'Sin configuración disponible';
            text('trafficMode', mode);
            document.getElementById('trafficMode').className = `badge ${p.mode === 'enforce' ? 'bg-success' : 'bg-secondary'}`;
            for (const [id, value] of [['trafficTransport', data?.transport_rejected], ['trafficBody', data?.body_rejected], ['trafficReadTimeouts', data?.read_timeouts], ['trafficLimited', data?.limited], ['trafficBanned', data?.banned], ['trafficObserved', data?.observed], ['trafficCapacity', data?.capacity_rejected], ['trafficStoreErrors', data?.store_errors]]) text(id, data ? number(value) : '—');
            text('trafficPolicy', p.rate ? `${number(p.rate)} solicitudes/s por IP · ráfaga de ${number(p.burst)} · archivos estáticos: ${number(p.static_rate)}/s por IP` : 'No se recibieron los límites de protección.');
            text('trafficBanPolicy', p.short_ban_seconds ? `Baneos de ${number(p.short_ban_seconds / 60)} y ${number(p.long_ban_seconds / 60)} minutos por abuso persistente` : 'Bloqueos temporales progresivos');
            text('trafficCapacityPolicy', p.request_capacity ? `Máximo ${number(p.request_capacity)} solicitudes y ${number(p.auth_capacity)} operaciones de autenticación simultáneas` : 'Límites de capacidad del servidor');
        },
    };
})();
