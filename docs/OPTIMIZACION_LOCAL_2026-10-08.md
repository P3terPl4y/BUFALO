# Pruebas locales de rendimiento y seguridad — 8 de octubre de 2026

BUFALO se inició exclusivamente en `http://127.0.0.1:33333`, con PostgreSQL
temporal en 55448 y Redis temporal en 56348. La instancia HTTP usa la base
`bufalo_local_test` y Redis DB 1; las suites usan `bufalo_auth_utf8_test`.
No se abrió el puerto público ni se usaron datos reales para estas pruebas.
El correo SMTP está desactivado en la instancia local.

## Cambios

- Las plantillas se recargan según `VIEWS_RELOAD`, cuyo valor predeterminado
  sigue `APP_DEBUG` (o el comportamiento anterior si no está definido).
  Con `APP_DEBUG=false` se reutilizan las plantillas también en modo local.
  Producción ya desactivaba la recarga; esta mejora no demuestra la causa del
  consumo histórico en producción.
- Los archivos de `public` se sirven después de las cabeceras de seguridad y
  antes de sesión/CSRF. No generan cookies ni accesos de sesión a Redis.
  Los archivos inexistentes continúan hacia las rutas y su autenticación.
- El contexto de cierre de Goravel ahora cierra Fiber en un máximo de 10 s.
  La instancia anterior ignoraba SIGTERM; la versión final liberó el puerto
  en 0,004 s y se reinició correctamente.

## Comparación

300 GET por ruta, 8 clientes, una conexión por solicitud, mismo host. El
generador comparte CPU con el servidor y las pruebas: son mediciones locales,
no capacidad máxima ni un benchmark comparable con un servicio Python distinto.

| Ruta | Antes, solicitudes/s | Después, solicitudes/s | p95 antes/después, ms |
| --- | ---: | ---: | ---: |
| `/css/style.css` | 447,9 | 887,0 | 40,59 / 12,82 |
| `/login` | 46,3 | 854,4 | 256,14 / 13,38 |
| `/register` | 49,4 | 781,1 | 238,40 / 16,26 |
| Ruta inexistente | 47,1 | 883,2 | 284,12 / 13,53 |

Pico RSS: 111,7 MiB antes y 83,1 MiB después. CPU acumulada de login para
300 solicitudes: 833 frente a 29 ticks de CPU. Registro: 780 frente a 40.
La diferencia principal es evitar volver a analizar todas las plantillas.
Una repetición con el binario final y 8 clientes obtuvo 694,2 solicitudes/s en
login y 624,1 en registro, con pico RSS de 81,9 MiB. La variación entre pasadas
refleja también la carga compartida del host.

La versión final completó 2.000 GET por cada una de cinco rutas, con 32
clientes: **10.000 solicitudes**, ningún error de transporte, ningún 5xx,
2.000 respuestas 404 esperadas. Pico RSS de 98,2 MiB. Login: 904,8 solicitudes/s;
registro: 701,4 solicitudes/s. Esto ensaya GET de formularios, no 10.000 altas.

## Funcionalidad y seguridad

- Suite completa `go test -p 1 ./... -count=1 -timeout=180s`: aprobada,
  incluidos feature, red-team de 100 usuarios, controladores y servicios.
- Login/registro: CSRF, rotación e invalidación de sesión, credenciales
  genéricas, cuenta deshabilitada, inyección SQL, límites de contraseña,
  rechazo de rol admin y confirmación falsificada.
- Confirmación: token de un uso, caducidad tras esperar bloqueos, concurrencia,
  imposibilidad de reemplazar el registro pendiente y transacciones atómicas.
- Redis real: cuotas atómicas, cuentas entre IPs/instancias y fallo cerrado.
- `deploy/local-smoke.py`: aprobado contra el binario final. Verifica cabeceras
  y ausencia de cookies en estáticos, health/readiness, páginas públicas,
  redirección de rutas protegidas, 404, CSRF 403 y login 429 después de 10 intentos.
- Compilación final y `git diff --check`: aprobados.
- Detector de carreras `go test -race -p 1 ./app/http/controllers
  ./app/http/middleware ./app/services -count=1 -timeout=180s`: aprobado.
  Controladores: 124,158 s; middleware: 1,699 s; servicios: 4,584 s.
  Se corrigió únicamente el timeout del cliente de test (1 s → 30 s): bcrypt
  instrumentado superaba el segundo. El coste de bcrypt de la aplicación no
  se redujo. Esa repetición usó `bufalo_auth_race_test`, también UTF-8 y aislada.

La primera suite se descartó por un clúster temporal SQL_ASCII; la repetición
aprobada usa una base UTF-8. También se descartó una medición que comenzó antes
de abrirse el puerto; el generador ahora espera `/healthz` y rechaza resultados
con errores de transporte.
La prueba HTTP espera ahora el arranque e identifica cada ejecución con un
correo distinto y una IP sintética de benchmark mediante el proxy loopback,
para no heredar cuotas anteriores. La comprobación final obtuvo diez respuestas
200 de credenciales inválidas y dos 429. Se verificó también la terminación
completa del proceso final por SIGTERM (0,001 s), seguida de nuevo arranque.

## Repetición y estado

Scripts versionables: `deploy/local-load.py` (solo loopback, máximo 2.000
solicitudes por ruta y 32 clientes) y `deploy/local-smoke.py` (modifica contadores
de login; usar solo la instancia aislada).

Instancia final: `/tmp/bufalo-local-final`; PID actual en `/tmp/bufalo-local.pid`;
log en `/tmp/bufalo-local-server.log`. Arranque aislado conservado en
`/tmp/start-bufalo-local.py`, con `LOCAL_BINARY=/tmp/bufalo-local-final`.
Los procesos temporales no tienen arranque automático tras reiniciar el host.
Las mediciones JSON se conservan junto a este informe.

Estas pruebas no demuestran resistencia frente a DDoS distribuido, saturación
del enlace o conexiones lentas. El consumo durante el incidente anterior no
se puede reconstruir con estas mediciones posteriores.
