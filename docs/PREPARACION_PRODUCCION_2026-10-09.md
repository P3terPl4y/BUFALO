# Validación y despliegue de producción — 9 de octubre de 2026

El candidato final quedó desplegado como servicio systemd en `127.0.0.1:3000`; Cloudflared llega al servicio directamente y no se usa Nginx. Su SHA-256 en disco y en el ejecutable activo es `7b7ac65db489e954fa1a399bd4eb9d0b11561557d8601ea0ace69bea3a6da499`. Las migraciones aditivas se aplicaron a la base real después del inventario y del respaldo privado. No se borraron datos comerciales.

## Evidencia de validación

- Suite completa: 207 pruebas y subpruebas aprobadas en 18 paquetes, cero fallos. La única prueba omitida inicialmente se ejecutó después contra PostgreSQL local aislado y aprobó.
- Detector de carreras: entrada HTTP, middleware, servicios, seguridad, perfiles y controladores aprobados; se repitió para la admisión y middleware CSRF finales.
- `go vet` y `git diff --check`: limpios.
- Navegador Firefox: 18 pasos de CRUD administrativo y 8 de registro/confirmación/preferencias; sin errores JavaScript.
- Carga aislada: 10 000 solicitudes con 32 trabajadores; 9948 rechazadas con 429. Las 24 comprobaciones de salud y de otra IP pasaron durante el pico. RSS máximo medido 75 012 KiB.
- Tras desplegar: 1000 solicitudes con 16 trabajadores; 957 rechazadas con 429. Salud y otra IP respondieron 200; RSS pasó de 50 344 a 55 612 KiB. El servicio no reinició.
- Cloudflared/HTTPS público: `/login`, `/register`, `/healthz` y `/readyz` respondieron 200.
- `govulncheck`: terminado, sin vulnerabilidades alcanzables; pico medido 1 505 234 944 bytes bajo un límite de 2 GiB.
- Restricciones de integridad PostgreSQL: migradas y validadas. Inventario previo: cero ratings inválidos y cero huérfanos detectados.

Los informes de pruebas se guardan en `storage/implementation-backup/`; el análisis de consumo de `govulncheck` está en `storage/security-scan/latest.json`. Las pruebas temporales se apagaron al final y sus directorios de datos permanecen conservados.

## Límites y pendientes de plataforma

El control del proceso limita admisión, conexiones, cuerpos, plazos, sesiones y concurrencia, y devuelve 429/503 al alcanzar capacidad. Ninguna configuración del servidor puede garantizar que un volumen ilimitado o distribuido jamás sature la red o el host; Cloudflare debe mitigar el tráfico antes de que llegue al túnel.

`GO-2026-5932` corresponde al paquete OpenPGP de `golang.org/x/crypto`, no importado por la aplicación y sin una versión corregida conocida en el inventario consultado. Se conserva `x/crypto` porque bcrypt lo requiere y CI prohíbe importar OpenPGP.

Docker y Podman no están instalados en este entorno, los user namespaces están denegados y no hay sudo disponible sin autenticación. El Dockerfile y el workflow de CI están preparados, pero no se afirma que la imagen se haya construido o desplegado. La plataforma PostgreSQL de producción aplica actualmente el mismo rol a ejecución y migraciones; separarlos necesita un administrador PostgreSQL. En staging se comprobó el rol web sin permisos DDL, pero no se extrapola ese resultado al host real.
