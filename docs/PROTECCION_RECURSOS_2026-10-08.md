# Protección y consumo de recursos — 8 de octubre de 2026

## Estado comprobado

Servidor de pruebas en 127.0.0.1:33333, DDOS_MODE=enforce, PostgreSQL aislado 55448 y Redis 56348. No se activó la instalación pública. Binario /tmp/bufalo-ddos-final; PID en /tmp/bufalo-local.pid. La configuración Docker/systemd conserva observe como primera etapa de despliegue.

- Presupuesto flexible Go de 128 MiB y GOMAXPROCS=2 por defecto; límites de trabajo de 64 solicitudes y 4 autenticaciones simultáneas, sin cola de espera.
- Login/registro/confirmación limitados a 16 KiB desde las cabeceras; uploads mantienen 6 MiB.
- Cuotas por IP en Redis, almacenamiento con expiración, baneos progresivos y caché local de hasta 4096 bloqueos. La caché no guarda permisos de acceso ni prolonga los vencimientos. Rechazos anteriores al logger para evitar una línea de disco por petición abusiva.
- Memoria RSS y contadores de protección visibles en ambos paneles administrativos. Contadores por proceso, no cifras globales de Cloudflare.
- Docker sin Nginx, puerto publicado solo en loopback, servicios de datos internos, volúmenes persistentes y límites de recursos.
- Script deploy/security-scan.sh para ejecutar govulncheck separado, con límite cgroup de 1 GiB incluyendo hijos, sin swap, objetivo Go 768 MiB, compilación serial y timeout de 15 minutos. Requiere govulncheck previamente instalado y systemd de usuario con control de memoria; no se ejecutó el análisis completo en esta continuación. Alcanzar el límite debe marcar fallo, no aprobación.

## Prueba de carga acotada

`python3 deploy/local-traffic-load.py`: 20 solicitudes normales a 10/s aceptadas. Después, 10.000 GET de login con una misma IP sintética, 32 clientes, 12,337 s, aproximadamente 810,5 solicitudes/s. Respuestas: 43 HTTP 200 y 9.957 HTTP 429. Sin errores de transporte ni 5xx. RSS previo 74,5 MiB y máximo observado durante la prueba 84,2 MiB. Health y una IP independiente conservaron HTTP 200. Un POST con Content-Length de 1 MiB, sin enviar cuerpo, recibió 413 inmediato.

La cifra de solicitudes/s mide sobre todo rechazo rápido bajo este generador y host compartido; no mide capacidad de operaciones de negocio ni resistencia a ataques distribuidos. El refill de tokens explica las solicitudes admitidas durante el inicio de la carga. No se ensayaron 100 millones de solicitudes/s ni saturación del enlace.

## Verificación funcional

- Suite completa: `go test -p 1 ./... -count=1 -timeout=240s`, aprobada con DB bufalo_auth_utf8_test, PostgreSQL 55448 y Redis 56348. Incluye controladores, feature y red-team (63,555 s para red-team).
- Detector de carreras: middleware y monitoring aprobados (2,550 s y 1,493 s), con Redis real.
- Smoke HTTP aprobado: CSRF, cookies/estáticos, rutas públicas/protegidas, diez intentos de login inválido aceptados y dos limitados.
- Firefox: login con administrador sintético y navegación por /admin/metrics y /admin/health. Ambos mostraron bloqueo activo y 9.858 solicitudes baneadas; API reportó 99 rechazos de cuota, cero fallos Redis y RSS 86,08 MiB tras navegar. RSS de navegador no pertenece al servidor.
- Migrador: ejecución idempotente correcta y base inaccesible produce salida no cero.
- Compilación Go, sintaxis JavaScript, sintaxis shell y git diff --check correctos.
- Compose 2.40.3: esquema e interpolación validados con credenciales sintéticas y rutas equivalentes. Docker Engine no está instalado; no se construyeron ni arrancaron contenedores. La unidad systemd tampoco se instaló ni activó.

## Incidente de govulncheck

El historial de la auditoría contiene `env TMPDIR=/home/peter/.cache/bufalo-auth-audit/gotmp go run golang.org/x/vuln/cmd/govulncheck@latest ./...`, ejecutado sin límites explícitos. El grafo actual de `go list -deps ./...` contiene 1057 paquetes. El usuario identificó govulncheck como el proceso eliminado, con anon-rss de aproximadamente 4.000.000 KiB. No se pudo acceder al registro global del kernel ni obtener un perfil del analizador; no se ha probado una fuga específica ni una dependencia responsable. Este incidente no atribuye esa RAM al servidor BUFALO.

Consulta docs/DEPLOY_PRODUCCION.md para configurar Cloudflared, migraciones, respaldo y transición observe → enforce.
