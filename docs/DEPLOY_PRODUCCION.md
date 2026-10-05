# Despliegue de BUFALO en producción

BUFALO escucha en `APP_HOST:APP_PORT` (por defecto `0.0.0.0:3000`). El proxy inverso debe terminar TLS y reenviar `Host` y `X-Forwarded-For`. No expongas el puerto de la aplicación directamente a Internet si el dominio ya usa un proxy.

## Preparación

1. Instala Docker Engine y el plugin Docker Compose en el host.
2. Copia `.env.production.example` como `.env.production` y configura secretos únicos, PostgreSQL y Redis alcanzables desde el contenedor. Usa `DB_SSLMODE=require` cuando PostgreSQL ofrezca TLS. No reutilices `.env` de desarrollo ni guardes `.env.production` en Git.
3. Construye la imagen: `docker compose build --pull`.
4. Antes de migrar, genera y valida un respaldo PostgreSQL fuera del contenedor.
5. Ejecuta las migraciones pendientes: `docker compose run --rm goravel artisan migrate`.
6. Inicia el servicio: `docker compose up -d`.
7. Comprueba `docker compose ps`, `docker compose logs --tail=100 goravel` y `curl -fsS http://127.0.0.1:3000/healthz`. La comprobación de salud solo confirma que el proceso HTTP responde; verifica también acceso autenticado, PostgreSQL, Redis y flujos de negocio.

Las imágenes no contienen archivos `.env`. Los avatares se guardan en el volumen persistente `bufalo_avatars`; haz respaldo de ese volumen junto con PostgreSQL. El contenedor corre como UID no privilegiado y usa filesystem de solo lectura salvo `/tmp` y el volumen de avatares.

## Actualización y recuperación

Haz respaldo de PostgreSQL y del volumen de avatares antes de actualizar. Construye la nueva imagen, ejecuta migraciones pendientes, y recrea el servicio con `docker compose up -d --force-recreate`. Si la migración falla, conserva logs y restaura el respaldo según el procedimiento operativo del host antes de volver a la versión anterior. La migración de perfiles de factura vuelve opcionales los IDs de empresa y atribuye facturas existentes a los perfiles asociados a su carga; revisa manualmente registros antiguos que no tengan una carga válida.

La aplicación necesita un servicio PostgreSQL y Redis persistentes. `docker-compose.yml` no crea ni publica esos servicios y el archivo `.env.production` debe apuntar a sus direcciones reales.

El endpoint `/admin/health` está restringido al rol administrador y no se almacena en caché. Muestra contadores HTTP desde el inicio del proceso, latencia media, p95 aproximado por intervalos, errores 5xx, memoria Go, goroutines y disponibilidad TCP de PostgreSQL/Redis. La comprobación TCP no valida credenciales ni ejecuta consultas; las métricas son locales al proceso y se reinician al reiniciarlo. Configura monitorización externa si necesitas alertas o históricos.

## Host Linux con systemd

El servicio incluido en `deploy/bufalo.service` permite administrar el proceso sin Docker. Ajusta `User`, `Group`, `WorkingDirectory`, `EnvironmentFile`, `ExecStart` y el origen CSRF al host. Compila con `go build -o bin/bufalo .`, instala la unidad en `/etc/systemd/system/bufalo.service`, y ejecuta `systemctl daemon-reload && systemctl enable --now bufalo`. El archivo de entorno debe ser legible por el usuario del servicio y no debe estar versionado. Actualiza con respaldo, compila el binario nuevo y ejecuta `systemctl restart bufalo`; revisa `systemctl status bufalo` y `journalctl -u bufalo`.
