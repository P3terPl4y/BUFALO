# BUFALO — plan y registro de preparación para producción

**Actualizado:** 6 de octubre de 2026  
**Versión estable de referencia:** `ddeea6f` (`Retira la app móvil y sus referencias externas`)  
**Entorno de esta revisión:** local, sin despliegue ni cambios en producción.

## Criterio de salida

BUFALO se considera listo para promover cuando los controles de código y las suites pasan en CI, una instalación de staging equivalente a producción demuestra el comportamiento de PostgreSQL, Redis, HTTPS, proxy y sesiones, y existe un procedimiento probado de respaldo y reversión. Una compilación correcta o un `/healthz` exitoso, por sí solos, no son evidencia suficiente.

## Mejoras aplicadas en esta revisión

- Se agregó `GET /readyz`, separado del liveness `GET /healthz`. La readiness ejecuta `SELECT 1` en PostgreSQL y `PING` en Redis bajo un plazo total de dos segundos. Responde 200 con `{"status":"ready"}` si ambos servicios contestan, o 503 con un mensaje genérico y `Cache-Control: no-store` si alguno falla. No publica errores, hosts ni credenciales. Ambos probes omiten la carga de sesión y CSRF para que Redis caído no oculte el liveness ni la respuesta degradada.
- El arranque con `APP_ENV=production` rechaza `APP_DEBUG=true` (y valores booleanos inválidos), `APP_KEY` distinto de los 32 caracteres exactos requeridos por Goravel y secretos conocidos de plantilla. La validación ocurre antes de `ensureAdminUser` para evitar escrituras bootstrap con configuración inválida. La clave de rate limit puede seguir heredando `APP_KEY`.
- Se añadieron pruebas unitarias de readiness saludable/degradada, deadline, respuesta sin datos internos y validación de configuración de producción. El middleware de métricas excluye `/readyz` de las métricas de tráfico de usuario.
- El pool PostgreSQL tiene límites configurables y conservadores por defecto (`max open=25`, `max idle=10`, inactiva=300 s, vida máxima=1800 s). El arranque productivo valida los rangos para impedir conexiones ilimitadas o configuraciones inválidas. El pool SQL descarta conexiones rotas y puede abrir conexiones nuevas en operaciones posteriores.
- Se añadió `DB_DSN` opcional para DSN PostgreSQL directo, incluidos hosts alternativos/failover compatibles con pgx. HA depende de la replicación y configuración del proveedor; no está activada ni demostrada sin un DSN real multi-nodo.
- Fallos de transporte, conexión SQL cerrada, `driver.ErrBadConn` y expiración de contexto se responden como 503 genérico. No se reintentan escrituras ni transacciones interrumpidas para evitar duplicar efectos confirmados cuya respuesta se perdió.
- Docker Compose usa `/readyz` como healthcheck de dependencias; PostgreSQL o Redis no disponibles marcan el contenedor como no saludable.
- Se corrigieron expectativas de `tests/redteam`: la prueba asumía que compartir empresa autorizaba ver y pagar cualquier factura. Ahora exige pertenencia al perfil asociado a la factura (o rol admin), acorde con el aislamiento previsto.
- Se actualizó la guía de despliegue y el ejemplo de configuración con las nuevas comprobaciones.
- El auto-registro ahora solicita confirmación por correo antes de crear usuarios, empresas o perfiles. Se almacena durante 24 horas un payload cifrado con `APP_KEY` en `pending_registrations`; el token de 256 bits se guarda solo como SHA-256. Abrir el enlace no confirma (protección ante escáneres); un POST explícito consume el token y crea cuenta, empresa y perfil en la misma transacción. Un token vencido/reutilizado se rechaza y una falla SMTP limpia la solicitud pendiente. Administradores que crean usuarios desde el panel conservan su flujo actual.
- Producción exige `APP_URL` HTTPS y `MAIL_HOST`/`MAIL_FROM_ADDRESS` válidos para no aceptar registros que no puedan verificarse. El envío real del proveedor SMTP todavía requiere prueba en staging.

## Matriz pendiente para promoción

| Prioridad | Comprobación | Evidencia requerida | Estado |
|---|---|---|---|
| P0 | Suite Go completa y compilación | CI limpio con `go test -p 1 ./... -count=1` y `go build`; adjuntar commit y salida | Aprobada tras estos cambios finales en PostgreSQL efímero; volver a exigir en CI para el commit de release |
| P0 | PostgreSQL de staging | Migraciones desde snapshot representativo, restricciones e índices, altas concurrentes, rollback ante fallo y consultas de `/readyz` | Pendiente en staging |
| P0 | Recuperación PostgreSQL | Interrumpir/reponer conexión, medir retorno a ready, verificar lecturas/escrituras posteriores y analizar operaciones en vuelo sin replay | Fallback de conexión con primer host cerrado probado en PG efímero; promoción/failback y RPO/RTO del proveedor pendientes en staging |
| P0 | Redis de staging | `PING`, sesión real, contador rate limit compartido entre dos instancias, expiración y prueba de indisponibilidad | Prueba aislada del contador aprobada; sesión y caída real pendientes en staging |
| P0 | HTTPS y proxy | Certificado y renovación, `Secure`/`HttpOnly`/`SameSite`, CSRF, `Host` y `X-Forwarded-For` del proxy real; intentar spoofing desde cliente | Pendiente de verificar topología real |
| P0 | Control de origen | Firewall impide llegar directamente al puerto de BUFALO; solo el proxy puede alcanzar el upstream | Pendiente en host/proveedor |
| P0 | Respaldo y reversión | Restaurar backup de PostgreSQL y archivos persistentes en entorno aislado; rollback del binario verificado | Pendiente, no inferir respaldo válido sin restaurarlo |
| P0 | SMTP y confirmación de correo | Enviar a buzón controlado en staging, validar link HTTPS, expiración, headers/reputación del proveedor y entrega/error | Flujos, expiración, replay y fallo SMTP simulado pasan localmente; entrega real pendiente |
| P0 | Rotación de credencial SMTP local | Revocar la credencial SMTP observada en la salida local de inspección y sustituirla por una guardada en el gestor de secretos; verificar que no llegó al repo ni al host de producción | Pendiente del operador |
| P1 | WAF/CDN y abuso | Límites antes del origen, protección L3/L4 del proveedor, reglas para login/registro y alertas de tasa | Pendiente de proveedor |
| P1 | Registro de usuarios | Verificación de correo o CAPTCHA administrado, recuperación y límites contra bloqueo dirigido por email | Confirmación por correo implementada; CAPTCHA, recuperación y revisión de abuso por correo siguen pendientes |
| P1 | Carga y capacidad | Definir SLO, concurrencia objetivo, duración, p95/p99, errores 5xx y umbrales de CPU/memoria/DB/Redis | Pendiente; no se ha ejecutado carga contra producción |
| P1 | Observabilidad | Alertas externas para readiness, 5xx, Redis/PostgreSQL, disco, memoria, backups y expiración TLS; retención y privacidad de logs | Pendiente en plataforma |
| P2 | Pruebas manuales de aceptación | Roles admin/publicador/chofer, permisos de datos, ciclo completo de carga/factura/dirección y UX móvil/navegadores soportados en staging | Suites HTTP pasan; aceptación visual y Redis real pendientes |

## Registro de ejecución

1. **06-oct-2026 — base de prueba:** la base de producción configurada no se usó. Se creó un clúster PostgreSQL efímero en `/tmp`, una base `bufalo_validation_test` con el usuario local `peter`, y se apagó el clúster tras finalizar.
2. **06-oct-2026 — ejecución previa:** `go test -p 1 ./... -count=1 -timeout=180s` detectó solo expectativas incorrectas en red-team: se esperaba que el chofer/carrier y publicadores de la empresa vieran y pagaran una factura sin asociación a su perfil. El código devolvió 403, que preserva el aislamiento. Se corrigió el test para aceptar acceso únicamente al perfil emisor o admin.
3. **06-oct-2026 — repetición:** la suite completa pasó después de corregir la expectativa. `go test ./app/http/middleware -race -count=1`, los paquetes sin DB y `go build` también pasaron en esa revisión.
4. **06-oct-2026 — cambios de readiness/configuración:** se implementaron `/readyz`, la omisión de sesión/CSRF para ambos probes, validación previa de configuración productiva y tests unitarios. `go test . ./app/monitoring ./app/http/middleware -count=1` pasó; `go build -p 1` pasó.
5. **06-oct-2026 — validación final previa a resiliencia DB:** `go test -p 1 ./... -count=1 -timeout=180s` pasó en todos los paquetes tras las mejoras de readiness, incluida la suite red-team de 100 usuarios y PostgreSQL temporal. `go test -race ./app/monitoring ./app/http/middleware -count=1`, `go build -p 1` y `git diff --check` pasaron. No se hicieron solicitudes contra producción.
6. **06-oct-2026 — resiliencia PostgreSQL añadida:** pool ajustable/validado, soporte para `DB_DSN`, respuestas 503 ante errores transitorios y healthcheck Docker basado en readiness. Se detectó y cerró un riesgo en la guardia destructiva: ahora también compara el nombre de base extraído del DSN efectivo con `DB_DATABASE`, y exige `_test` en ambos. Un DSN con primer host cerrado y segundo host temporal activo pasó pruebas de servicio y la suite completa `go test -p 1 ./... -count=1 -timeout=180s`. También pasaron pruebas dirigidas del parser y la guardia. La prueba de failover del proveedor y su failback/RPO/RTO quedan en staging.
7. **06-oct-2026 — verificación de cierre:** `go build -p 1 -o /tmp/bufalo-db-resilience-final .`, `go test -race -p 1 ./app/dbresilience ./app/http/middleware ./app/monitoring -count=1` y `git diff --check` pasaron. Un intento previo de ampliar `-race` a `./tests` chocó con el límite de espacio temporal al compilar dependencias; se liberó el clúster temporal propio y la compilación normal completa pasó después. No se interpretó ese intento como prueba aprobada. El `pgx/v5` usado por el parser se declaró como dependencia directa. No se modificó producción.
8. **06-oct-2026 — registro con verificación de email:** se agregó una migración y almacenamiento cifrado temporal; la entrega SMTP debe estar configurada. Un GET del link solo muestra una página de confirmación para evitar que escáneres de correo activen cuentas; el POST confirma, crea datos atómicamente y consume el token. Pasó la suite completa con PostgreSQL aislado y una base `_test` independiente por paquete que ejecuta migraciones; feature y red-team (100 usuarios) incluidos. Se añadieron después los casos de expiración/replay y falla SMTP, y el paquete de controladores pasó otra vez en su propia base aislada. `go build` pasó y los paquetes `dbresilience`, middleware y monitorización pasaron `-race`; el root bajo `-race` volvió a superar la cuota temporal al enlazar. No se enviaron correos reales ni se tocó producción.

## Reglas para la siguiente etapa

- No desplegar desde un árbol de trabajo mezclado: la revisión local contiene otros cambios previos además de este endurecimiento. Preparar un commit/release acotado y revisar su diff completo antes de publicar.
- Desplegar primero a staging, probar `/readyz` saludable y degradado, flujos autenticados y caída de Redis. El fail-closed actual hace que login/registro devuelvan 503 si Redis falla; acordar operación y alertas para ese escenario.
- La validación de secretos detecta longitud insuficiente y marcadores de plantilla; no puede demostrar entropía real. Genera claves criptográficamente aleatorias y guárdalas en el gestor de secretos del host.
- No se repiten transacciones automáticamente. Para HA, confirmar RPO/RTO con el proveedor y probar las operaciones concurrentes e idempotency keys donde un cliente pueda reenviar una solicitud cuyo resultado haya quedado ambiguo.
- Aplicar carga solo contra staging con datos sintéticos. No generar tráfico adversarial ni pruebas DDoS contra producción.
- La app mitiga abuso a nivel HTTP; no puede detener por sí sola una botnet o saturación de red. La protección volumétrica depende del proveedor/WAF y de bloquear el origen.
