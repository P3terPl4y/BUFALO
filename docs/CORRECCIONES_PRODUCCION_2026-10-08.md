# Correcciones y validación de producción

Este informe corresponde a la implementación y validación del 8 de octubre. La producción está activa mediante systemd de usuario en `127.0.0.1:3000`, con salud interna en `127.0.0.1:3001`. Las evidencias se conservan en `storage/implementation-backup/`.

## Resultados de aceptación

- Compilación: Go 1.26.8, CGO desactivado, `-trimpath -ldflags='-s -w'`. SHA-256 del archivo validado, instalado y ejecutado: `19ec79ab3895d05ce630b35bfacbfb3549320336bda9f6293f6306c3ff8d0ced`. Binario de 58.917.026 bytes frente a 109.654.653 del anterior: reducción del 46,3 %. Se conserva el anterior para recuperación.
- Suite completa aprobada: pruebas unitarias, integración, funcionales y redteam. Detector de carreras aprobado en ingreso HTTP, middleware, métricas, sesiones, correo y resiliencia de base de datos. Evidencias: `acceptance-complete.log`, `race-complete.log`, `compatibility-tests.log`.
- `go vet ./...` terminó con código 0 y sin diagnósticos (`vet-acceptance.log`); `git diff --check` también pasó. Se eliminaron 200.106.470 bytes de binarios de validación obsoletos, conservando la versión desplegada, la copia con símbolos y el respaldo de recuperación.
- Carga local aislada: 10.000 solicitudes, 32 trabajadores, 854 solicitudes/s; 45 respuestas 200 y 9.955 respuestas 429 esperadas. Sin errores de conexión; salud y otra identidad siguieron respondiendo 200. RSS máximo 70.100 KiB (68,46 MiB). No representa capacidad de tráfico útil ni resistencia a 100 millones de solicitudes/s. Evidencia: `final-load.json`.
- Ingreso HTTP: límites de cuerpo aplicados también a alias; cuerpos lentos agotaron su plazo; 64 cuerpos lentos no impidieron salud ni acceso desde otra identidad. Evidencias: `final-ingress.json`, `final-smoke.json`.
- Firefox: login del administrador temporal aislado, panel de métricas y salud, consumo de govulncheck y rechazos visibles. Foto PNG válida de 5.242.752 bytes aceptada; SVG inválido rechazado. Cuenta, credenciales y foto de prueba retiradas. Evidencia: `final-panel-verified.json`.
- Antes de publicar: candidato con configuración real de producción, cookies `__Host-` seguras, rutas 200, administración protegida 303 y reinicio automático comprobado. Redis de tráfico sin memoria: readiness 503, liveness 200, nuevas identidades 503; recuperación 200 al restaurar capacidad. Evidencia: `deployment-final.json`.
- Después de publicar: salud, disponibilidad, login, registro y CSS 200; administración anónima 303; entorno production y debug desactivado comprobados en el proceso. Cuerpo excesivo de login y cuerpo en salud rechazados con 413; resumen de rechazos confirmado en journal. Sin reinicios. Memoria del grupo observada: 16.310.272 bytes, pico 17.244.160 bytes. Datos conservados: 7 usuarios, 4 cargas y 4 facturas. Evidencia: `postdeploy-final.json`. Estas mediciones breves no sustituyen observación prolongada.
- Govulncheck del binario de auditoría con símbolos, construido con el mismo código y dependencias: aprobado, sin vulnerabilidades conocidas que afecten al binario; pico de grupo 114.933.760 bytes (109,6 MiB), CPU 2,48 s, duración 4,14 s, límite 1 GiB. Evidencia: `scan-final-binary.json`. El archivo distribuido elimina símbolos; el análisis usa su copia de auditoría para conservar precisión.
- Repetición del análisis completo del grafo de llamadas del código fuente, finalizada el 8 de octubre a las 18:59:17 UTC: **aprobada**, código 0, cero vulnerabilidades que afecten al código y cero en paquetes importados. Pico 1.498.775.552 bytes (1,40 GiB), límite 2 GiB, duración 85,303 s y CPU 75,543 s. Se informa una vulnerabilidad en un módulo requerido cuyo código vulnerable no aparece llamado; esto no equivale a que todos los módulos estén libres de avisos. Producción respondió 200 durante y después, sin reinicios. Evidencias: `scan-source-2g-completed.json` y `.log`. Los intentos anteriores agotaron el límite (`scan-final-source.json`); este nuevo resultado completa el análisis antes pendiente. No se ha aislado la causa exacta de la diferencia de consumo entre ejecuciones. El panel muestra el nuevo resultado source-symbols.

## Pendientes y límites de validación

- Acceso público: cloudflared está activo, pero la resolución de `bufalo.duohnson.com` falla desde este host (`curl: Could not resolve host`). Solo se certifica acceso directo local; falta comprobar DNS y HTTPS públicos desde una red que resuelva el dominio.
- Docker: configuración Compose validada y roles probados en PostgreSQL aislado; no hay motor Docker disponible para construir y arrancar la imagen. Ese despliegue no está certificado.
- PostgreSQL existente: falta separar el usuario web y revocar CREATEDB/propiedad heredada mediante una cuenta administradora. El usuario actual no puede hacerlo y sudo requiere autorización interactiva. No se cambiaron esos privilegios.
- Entrega SMTP real a un buzón controlado pendiente; los plazos y comportamiento SMTP sí tienen pruebas con servidor controlado, sin enviar mensajes a terceros.

## Compatibilidad y mantenimiento

Se actualizaron las dependencias vulnerables. Goravel 1.18.0 requiere una adaptación a la API de logs OpenTelemetry 0.21: el snapshot local `third_party/goravel-framework` conserva la licencia y modifica únicamente dos archivos de ejecución del mapeo de logs. Sus pruebas específicas pasaron. Consultar `third_party/goravel-framework/BUFALO_COMPATIBILITY.md` antes de actualizar el framework. Los proveedores de instrumentación no utilizados se retiraron del arranque; las métricas propias del administrador siguen funcionando.

Los temporales de herramientas Go usan `storage/_build-tmp`, excluido de la enumeración de paquetes por Go. La antigua ruta `storage/build-tmp` contiene un marcador de módulo para que los restos de compilación CGO no entren en `go vet ./...`.

## Controles implementados

- Transporte HTTP del mismo proceso: 256 conexiones, cabeceras con plazo de 3 segundos, 64 solicitudes simultáneas, 8 por IP, admisión global de 200 solicitudes/s con ráfaga de 400. Salud interna en loopback, puerto de aplicación + 1.
- Lectura del cuerpo antes del adaptador Fiber: autenticación 16 KiB, formularios 256 KiB, foto de perfil 6 MiB y máximo 2 cargas simultáneas. Los alias de mayúsculas y barra final reciben el mismo límite. Lectura de formulario 5 segundos; foto 20 segundos.
- Política por IP: 20 solicitudes/s, ráfaga de 40; recursos estáticos 100/s, ráfaga de 200. Baneo progresivo de 1 minuto y 20 minutos por abuso persistente. No equivale a limitar conexiones TCP a cinco por segundo. El máximo de conexiones y de solicitudes activas son controles independientes.
- Redis de tráfico y sesiones independientes; memoria 32/64 MiB y `noeviction`. Registro de hasta 20.000 identidades y 10.000 sesiones nuevas. Sesión máxima 4 KiB. Las sesiones existentes pueden actualizarse aunque se alcance la capacidad de nuevas sesiones.
- Comprobación de disponibilidad mediante SELECT 1 y escritura con TTL en ambos Redis. Las sondas públicas no multiplican el trabajo: una comprobación en curso y caché de un segundo. Liveness no requiere dependencias.
- Normalización y validación de APP_ENV; producción y staging siempre usan cookies seguras. Producción no acepta depuración ni secretos de ejemplo.
- Login: 4 operaciones simultáneas. Registro: 2, con capacidad independiente. SMTP con conexión de 3 segundos y plazo total de 15 segundos; TLS obligatorio para autenticación remota.
- PostgreSQL: consulta web 5 segundos, conexión 5 segundos y transacción inactiva 10 segundos. Migraciones con bloqueo asesor compartido y plazo de sentencia de 120 segundos.
- Limpieza de registros de confirmación vencidos en lotes de 256 cada minuto, sin bloquear filas en uso ni eliminar tokens vigentes.
- Un solo binario estático en Docker, sin símbolos de depuración. Proveedores AI/OpenAI/gRPC que no se usaban retirados del arranque. Se conserva el soporte CLI y de pruebas del framework.
- Panel de administrador: rechazos de transporte, cuerpos excesivos, plazos, cuotas, baneos y capacidad. Registro en memoria de hasta 128 eventos con identidad HMAC, sin IP directa, URL, cuerpo ni credenciales. El diario de systemd recibe un resumen de rechazos como máximo una vez por segundo. Son indicios de abuso o sobrecarga, no atribución de un ataque.
- Analizador: `deploy/security-scan.sh` ejecuta el monitor Python y govulncheck fijado, con cgroup, límite de memoria, sin swap, CPU 100%, 64 tareas y máximo 15 minutos. Guarda el consumo agregado del grupo, CPU, duración y resultado en `storage/security-scan/latest.json`, visible solo en el panel protegido. Los últimos cinco informes permanecen en ese directorio. Un análisis interrumpido no se considera aprobado.

## Despliegue

El host usa cloudflared existente hacia 127.0.0.1:3000, sin Nginx. La versión de producción es la misma compilación validada antes de publicar y se verifica mediante SHA-256 del archivo y `/proc/PID/exe` después del arranque. systemd limita BUFALO a 512 MiB, sin swap, CPU 150% y 128 tareas; Go tiene objetivo de heap de 128 MiB y dos procesadores. El objetivo de heap no limita toda la memoria del proceso.

El usuario tiene `Linger=yes`; los servicios de usuario habilitados pueden arrancar sin una sesión interactiva. Las dos instancias privadas de Redis tienen unidades propias, persistencia AOF y memoria limitada. El Redis compartido del equipo no se altera. La base y uploads se respaldaron antes del despliegue y el respaldo se restauró en una base aislada.

Docker Compose valida su configuración y su script de roles pasó en PostgreSQL aislado: CRUD permitido, DDL denegado para el usuario web y ningún superusuario de aplicación. El motor Docker debe estar instalado y accesible para validar realmente la construcción y el arranque de contenedores. La inicialización de roles solo se ejecuta en un volumen PostgreSQL nuevo; migrar un volumen existente requiere pasos explícitos y respaldo, nunca borrar el volumen.

La cuenta PostgreSQL del host es no superusuario, pero conserva CREATEDB y propiedad del esquema heredadas. Revocar esos privilegios o crear una cuenta web separada exige una cuenta administradora de PostgreSQL; no se debe confundir la configuración de roles probada para Docker con los permisos del host existente.

## Alcance de las pruebas de carga

Las cargas se dirigen exclusivamente al puerto local 33333, con base aislada. Se prueban 10.000 peticiones, cuerpos excesivos, conexiones lentas, cuotas y recuperación. No se afirma resistencia a 100 millones de peticiones por segundo: ningún límite dentro de la aplicación evita por sí solo saturar el enlace o el núcleo del sistema. Cloudflare debe absorber el tráfico volumétrico antes del origen.
