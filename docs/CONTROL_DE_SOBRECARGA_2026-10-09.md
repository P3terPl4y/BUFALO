# Control de sobrecarga

El servidor limita el trabajo admitido; no puede garantizar disponibilidad ante tráfico ilimitado que agote la red o los recursos del host.

- Admisión sin cola: máximo 64 peticiones en ejecución, 8 por IP y 2 subidas de archivos.
- Presupuesto global de 200 peticiones por segundo, con ráfaga de 400; exceso rechazado antes de leer el cuerpo y ejecutar controladores.
- Las IP con baneo en caché se rechazan antes de gastar ese presupuesto o consultar Redis. La caché tiene un máximo de 4096 entradas.
- Límites de cuerpos: 16 KiB para login y registro, 256 KiB generales y 6 MiB para fotos. Plazos de lectura y conexiones acotadas.
- Historial de rechazos limitado a 128 entradas, identidades pseudonimizadas y escritura de logs agregada una vez por segundo.
- Los listados de choferes por empresa se consultan por páginas; consultar una empresa no precarga todos sus choferes.

El exceso recibe 429 o 503 con Retry-After; rechazar carga es un comportamiento esperado. El límite de RAM del servicio es un techo de contención, no una garantía de que no alcance OOM. La defensa local necesita protección de entrada en Cloudflare para ataques que superen la capacidad de red.

Pruebas de regresión: TestGlobalGateDoesNotQueue, TestSlowBodiesDoNotBlockHealthOrAnotherIP y TestCachedAbuseCannotSpendAdmissionBudget. El script deploy/stage-traffic-load.py ejecuta 10 000 solicitudes contra staging aislado y comprueba salud y una IP independiente durante el pico, además de medir RSS. No es una prueba de millones de solicitudes por segundo.

## Medición del candidato

En la última ejecución registrada en `storage/implementation-backup/production-stage-load-final.json`: 10 000 peticiones, 32 trabajadores, 15,32 segundos, 9948 rechazos 429 y 52 respuestas 200. Las 24 comprobaciones de una identidad independiente y de salud realizadas durante el ataque respondieron 200; el par más lento tardó 254,4 ms. RSS pasó de 61 680 a 75 012 KiB. Se ejecutó contra el candidato aislado de producción, no contra usuarios reales. Esta medición no representa un límite máximo de capacidad ni una garantía para un ataque distribuido. Binario SHA-256: `7b7ac65db489e954fa1a399bd4eb9d0b11561557d8601ea0ace69bea3a6da499`.

La prueba de navegador bajo esa carga completó 18 comprobaciones del CRUD administrativo sin errores de JavaScript. Las pruebas con detector de carreras de entrada HTTP, middleware, servicios, red team, perfiles y controladores terminaron correctamente. Los informes individuales conservan sus resultados y duración.
