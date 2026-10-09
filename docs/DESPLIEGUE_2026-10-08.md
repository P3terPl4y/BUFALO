# Despliegue local de BUFALO — 8 de octubre de 2026

Se desplegó el árbol de trabajo actual, con HEAD `ddeea6f` y cambios locales,
por solicitud expresa del usuario. No se creó un commit ni se hizo push.

## Cambios incluidos

- Confirmación de registro por correo con tokens de un uso, payload cifrado,
  expiración de 24 horas y creación transaccional de usuario/empresa/perfil.
- Límites de autenticación compartidos en Redis y comportamiento 503 ante caída.
- Readiness PostgreSQL/Redis, configuración productiva validada, pool DB y DSN.
- Mejoras de alta de usuarios y creación/edición administrativa de cargas/facturas.
- Páginas de error, códigos HTTP, diálogo de aceptación de cargas y estilos.
- Nuevas pruebas y documentación del estado del proyecto.

## Preparación y ejecución

Se compiló `bin/bufalo.next` y luego se sustituyó `bin/bufalo`. Se añadió
`deploy/run-production.py` para exportar el archivo `.env` al entorno del proceso;
la validación de `main.go` lee variables del proceso directamente. El lanzador
admite las entradas KEY=value y valores entre comillas usadas en este entorno;
no es un intérprete shell ni implementa interpolación de variables.

En `.env` se configuraron `MAIL_FROM_ADDRESS` con la cuenta Gmail SMTP ya existente,
`MAIL_FROM_NAME=BUFALO` y `APP_URL=https://bufalo.duohnson.com`. No se imprimieron
credenciales. La configuración anterior quedó respaldada.

Respaldo: `storage/deployments/20261008-183533/`, con dump PostgreSQL en formato
custom, archivo de uploads, configuración anterior y binario anterior activo.
`pg_restore --list` validó que el archivo es legible; no se efectuó una restauración.

Las 12 migraciones anteriores estaban aplicadas. Se ejecutó únicamente la pendiente
`20261006000001_create_pending_registrations_table`, con resultado satisfactorio.
Goravel no admite `migrate --force`; ese intento no aplicó cambios y se usó `migrate`.

El antiguo proceso `/tmp/bufalo-release` (PID 1868673) no terminó con SIGTERM;
se comprobó su identidad y se terminó con SIGKILL para liberar el puerto solicitado.
El primer intento de screen desacoplado no persistió; se inició correctamente con
`screen -DmS` bajo PTY. Servicio actual: sesión `bufalo-port3000`, binario
`/home/peter/BUFALO/bin/bufalo`, PID observado 3409590, escucha `127.0.0.1:3000`.
No existe una unidad systemd BUFALO instalada; screen conserva el proceso al
desconectarse, pero no proporciona arranque automático después de reiniciar el host.

Log: `storage/deployments/current-server.log`.

## Validación actual

- `go build -p 1`: aprobado.
- Nueve paquetes con pruebas sin PostgreSQL de integración: aprobados; `routes`
  compila y no contiene tests. `git diff --check`: aprobado.
- `/healthz` y `/readyz`: 200; readiness devuelve `{"status":"ready"}`.
- `/`, `/login`, `/register`, `/register/confirm`: 200.
- `/admin/users` sin sesión: 303; ruta inexistente: 404.
- Login entrega cookie `__Host-session` Secure y HttpOnly.

No se ejecutó la suite completa que trunca tablas, no se enviaron correos reales,
no se probaron recorridos autenticados ni la llegada por HTTPS público. Estas
comprobaciones no declaran resueltos los pendientes de staging ni el hallazgo
de privacidad de `/home` descrito en `MAPA_MANTENIMIENTO_2026-10-08.md`.

Para repetir el arranque desde el repositorio: `python3 deploy/run-production.py`.
Antes de reiniciar, identificar y detener el proceso actual. Para revertir el
binario se conserva `bufalo.previous` en el respaldo; la tabla nueva es aditiva,
por lo que no es necesario borrarla para volver al binario anterior. La reversión
de configuración debe considerar los valores del archivo `env.previous`.
