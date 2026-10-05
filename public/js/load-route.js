(function () {
    'use strict';

    const serviceURL = 'https://router.project-osrm.org/route/v1/driving/';
    const routeStyle = { color: '#D5001C', weight: 5, opacity: 0.9 };

    async function draw(map, pickup, dropoff, statusNode) {
        const direct = L.polyline([pickup, dropoff], {
            color: '#D5001C', weight: 3, opacity: 0.65, dashArray: '8, 8',
        }).addTo(map);
        const timeout = new AbortController();
        const timer = window.setTimeout(() => timeout.abort(), 8000);
        try {
            const coordinates = `${pickup[1]},${pickup[0]};${dropoff[1]},${dropoff[0]}`;
            const url = `${serviceURL}${coordinates}?overview=full&geometries=geojson&steps=false&alternatives=false`;
            const response = await fetch(url, { signal: timeout.signal, headers: { Accept: 'application/json' } });
            if (!response.ok) throw new Error('El servicio de rutas no está disponible.');
            const result = await response.json();
            const route = result && result.code === 'Ok' && result.routes && result.routes[0];
            if (!route || !route.geometry || !Array.isArray(route.geometry.coordinates) || route.geometry.coordinates.length < 2) {
                throw new Error('No se encontró una ruta por carretera para estas coordenadas.');
            }

            direct.remove();
            const road = L.geoJSON(route.geometry, { style: routeStyle }).addTo(map);
            map.fitBounds(road.getBounds(), { padding: [50, 50], maxZoom: 12 });
            if (statusNode) {
                const km = (route.distance / 1000).toFixed(1);
                const minutes = Math.max(1, Math.round(route.duration / 60));
                statusNode.textContent = `Ruta de conducción más rápida estimada: ${km} km · ${minutes} min. Coordenadas consultadas en OSRM/OSM; no incluye tráfico en tiempo real.`;
            }
        } catch (error) {
            if (statusNode) {
                statusNode.textContent = `${error.name === 'AbortError' ? 'La consulta de ruta tardó demasiado.' : error.message} Se muestra la línea directa entre los puntos.`;
            }
        } finally {
            window.clearTimeout(timer);
        }
    }

    window.BufaloRoadRoute = { draw };
})();
