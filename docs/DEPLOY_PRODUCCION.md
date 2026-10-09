# Despliegue de BUFALO en producción

## Docker y Cloudflared

El Compose incluye BUFALO, PostgreSQL y dos Redis persistentes. No utiliza Nginx.
Solo publica `127.0.0.1:3000`; el Cloudflared del host conecta directamente a ese puerto.
PostgreSQL y Redis no publican puertos. Se requiere Docker Engine con Compose v2.

1. Copia `.env.production.example` a `.env.production` y configura APP_KEY de 32 caracteres, APP_URL HTTPS, origen CSRF, SMTP y contraseñas únicas DB_PASSWORD, DB_ADMIN_PASSWORD, DB_MIGRATION_PASSWORD, REDIS_PASSWORD y RATE_REDIS_PASSWORD. Protege el archivo con permisos 600.
2. Si hay datos existentes, respalda y restaura PostgreSQL y uploads en los volúmenes nuevos antes de cambiar el túnel. Este Compose crea una base independiente: no importa automáticamente los datos de una instalación anterior. No ejecutes `down -v` sobre datos que necesites conservar.
3. Valida y construye:

```sh
docker compose --env-file .env.production config --quiet
docker compose --env-file .env.production build --pull
docker compose --env-file .env.production up -d
```

El servicio `migrate` ejecuta las migraciones y debe finalizar con éxito antes de iniciar BUFALO. El binario de migración devuelve un código distinto de cero al fallar. Revisa los resultados:

```sh
docker compose --env-file .env.production ps -a
docker compose --env-file .env.production logs --tail=100 migrate goravel
curl -fsS http://127.0.0.1:3000/healthz
curl -fsS http://127.0.0.1:3000/readyz
```

Usa siempre `--env-file`: las variables que Compose sustituye no se cargan solo con el atributo `env_file` del servicio.
`/readyz` ejecuta SELECT 1 y escrituras temporales con credenciales en ambos Redis, con trabajo acotado y caché de un segundo; `/healthz` comprueba que el proceso responde.
Los volúmenes `postgres_data`, `redis_data`, `redis_traffic_data`, `uploads` y `storage` requieren respaldo según su contenido.
La aplicación ejecuta como UID 10001, sin capacidades, con raíz de solo lectura y un /tmp acotado.

La red usa `172.30.113.0/24`. Si hay conflicto, cambia tanto la subred/gateway como TRUSTED_PROXY_IPS.
Solo se acepta CF-Connecting-IP desde loopback o el gateway configurado. Verifica en staging que el tráfico del túnel llega desde ese gateway y conserva la IP cliente antes de activar bloqueos; no confíes en proxies arbitrarios.

## Límites y métricas

La configuración predeterminada usa DDOS_MODE=enforce. Revisa `/admin/health` y `/admin/metrics`; usa observe temporalmente solo para ajustar límites al tráfico legítimo. Observe contabiliza decisiones sin bloquear por el limitador global; las cuotas previas de autenticación siguen activas.

La política dinámica admite 20 peticiones/s/IP con una ráfaga de 40. Los estáticos tienen un presupuesto separado de 100/s y ráfaga de 200. Cien rechazos en un minuto producen un bloqueo de un minuto; el tercero dentro de 30 minutos dura 20 minutos. Los intentos durante un bloqueo no prolongan su vencimiento. Redis coordina instancias; una caché local de hasta 4096 bloqueos evita consultar Redis por cada petición bloqueada.

La concurrencia de trabajo está limitada a 64, a 8 por IP, la de login a 4 y la de registro a 2; al agotarse se responde 503 sin cola. Login, registro y confirmación tienen cuerpo máximo de 16 KiB, aplicado al recibir las cabeceras. Los formularios generales admiten 256 KiB; las fotos conservan 6 MiB con dos cargas simultáneas. En enforce, Redis inaccesible provoca 503; no se permite eludir las cuotas. Redis usa noeviction para evitar borrar silenciosamente sesiones o bloqueos, y puede rechazar escrituras al llenarse.

El panel muestra RSS del proceso, heap Go y contadores de límites, solicitudes baneadas, observación, capacidad y fallos del almacén. Los contadores son por proceso y se reinician con él; los bloqueos permanecen en Redis hasta vencer. No incluyen ataques detenidos por Cloudflare.

GOMEMLIMIT=128MiB es un objetivo flexible del runtime, no un límite total de RAM. Docker impone 512 MiB a BUFALO, 512 MiB a PostgreSQL 192 MiB a Redis de sesiones y 128 MiB al de tráfico; son máximos, no memoria reservada. Estos límites deben ajustarse con mediciones reales. Un origen no puede evitar la saturación de su enlace ni garantizar 100 millones de peticiones/s; la protección de red debe aplicarse también en Cloudflare.

## Actualizaciones

Respalda PostgreSQL y uploads, construye la nueva imagen y ejecuta `docker compose --env-file .env.production up -d --force-recreate`. Comprueba la salida del migrador y readiness. No reviertas una imagen sin comprobar que su esquema sigue siendo compatible.

En producción el proceso rechaza `APP_DEBUG=true`, `APP_KEY` que no tenga exactamente 32 caracteres, secretos de plantilla y configuración incompleta del pool/base de datos antes de ejecutar el alta bootstrap opcional. Mantén `APP_DEBUG=false`, reemplaza todos los valores de plantilla y no compartas claves entre entornos.

El registro público crea una solicitud temporal cifrada y envía un enlace de confirmación válido por 24 horas. Los registros de usuario, empresa y perfil solo se escriben después del POST de confirmación. Configura `APP_URL` en HTTPS, `MAIL_HOST`, `MAIL_PORT` y `MAIL_FROM_ADDRESS`; producción no inicia si faltan la URL o el servidor/emisor. Prueba la entrega con un buzón controlado en staging antes de abrir el registro.

El pool PostgreSQL usa límites configurables: por defecto 25 conexiones abiertas, 10 inactivas, reciclaje de conexiones inactivas a los 300 segundos y vida máxima de 1800 segundos. Ajusta `DB_POOL_*` según el límite del servidor multiplicado por el número máximo de instancias. El driver descarta conexiones rotas y vuelve a conectar en operaciones posteriores; una solicitud ya interrumpida puede fallar y BUFALO no repite automáticamente escrituras.

Si el servicio PostgreSQL ofrece HA con varios nodos, `DB_DSN` permite entregar un DSN pgx con hosts de respaldo y `target_session_attrs=read-write`; al reconectar, el driver puede elegir un nodo disponible que acepte escrituras. Esto requiere replicación/failover correctos en PostgreSQL o en el proveedor. Debe probarse la caída y promoción de nodos en staging; configurar varios hosts no crea por sí mismo una réplica ni garantiza failover sin pérdida de datos.

## Host Linux con systemd

`deploy/bufalo.service` mantiene el servidor en loopback y aplica GOMEMLIMIT=128MiB, dos procesadores Go, MemoryHigh=384M y MemoryMax=512M. Ajusta rutas, usuario y entorno al host antes de instalar la unidad. Para servicios de usuario se incluye `deploy/bufalo-user.service`; requiere instancias privadas de Redis y linger habilitado. El estado efectivo del despliegue se registra en el informe de correcciones.

## Análisis de vulnerabilidades separado

Ejecuta el análisis en CI o en una máquina de desarrollo, fuera del servicio público. Instala previamente una versión revisada de govulncheck y ejecuta:

```sh
bash deploy/security-scan.sh
```

Por defecto comprueba paquetes importados, sin construir el grafo completo de llamadas. Para analizar llamadas usa `BUFALO_SCAN_MEMORY_BYTES=2147483648 bash deploy/security-scan.sh ./...` en un equipo con memoria disponible; para un binario conservando símbolos, `bash deploy/security-scan.sh -mode=binary ruta/binario`. El modo y su alcance se muestran en el informe.

El script requiere systemd de usuario y un controlador de memoria disponible. Impone 1 GiB al conjunto del analizador y sus hijos, sin swap; GOMEMLIMIT=512MiB, GOMAXPROCS=2, compilación serial y máximo de 15 minutos. Si systemd no puede aplicar los límites, falla: no continúa sin protección. Una terminación por límite o timeout es un análisis incompleto y debe tratarse como fallo, nunca como ausencia de vulnerabilidades. El monitor registra el estado, memoria máxima del cgroup, CPU y duración en `storage/security-scan/latest.json`, y conserva cinco informes de salida. El panel de administrador muestra el último resultado; no trata un fallo como aprobación.

Consulta [el informe de correcciones](CORRECCIONES_PRODUCCION_2026-10-08.md) para los cambios de transporte, versiones parcheadas, evidencias y limitaciones reales del host.

El análisis completo de llamadas terminó correctamente en la repetición del 8 de octubre: pico 1,40 GiB, límite 2 GiB y duración 85,303 segundos. Intentos anteriores agotaron 2 GiB; conserva el aislamiento y trata cualquier interrupción como fallo. El análisis del binario con símbolos también terminó correctamente. No interpretes el modo de paquetes como análisis completo de llamadas.

`go.mod` usa un snapshot local de Goravel para compatibilidad con la API de logs parcheada. Docker copia ese módulo antes de descargar dependencias. Revisa `third_party/goravel-framework/BUFALO_COMPATIBILITY.md` al actualizar versiones; conserva sus pruebas y licencia.
