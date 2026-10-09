# Endurecimiento de registro, concurrencia y tráfico abusivo

**Fecha:** 6 de octubre de 2026  
**Alcance:** rutas públicas de inicio de sesión/registro, creación concurrente de cuentas y mitigación operativa de DDoS.  
**Producción:** no se desplegó este cambio ni se ejecutaron pruebas de carga contra producción.

## Hallazgos iniciales

1. Inicio de sesión y registro usaban un limitador fijo de cinco solicitudes por minuto e IP con almacenamiento en memoria. El contador se reiniciaba al reiniciar el proceso y no coordinaba varias instancias.
2. El alta de empresa, usuario y perfil se ejecutaba con escrituras separadas y borrados compensatorios. Un fallo de proceso podía dejar registros parciales.
3. La unicidad de email estaba protegida por índice SQL, pero la aplicación no normalizaba consistentemente mayúsculas y espacios antes de verificar/guardar.
4. Ningún middleware de aplicación puede detener un DDoS volumétrico que sature el enlace o el host antes de que la solicitud alcance BUFALO.

## Cambios realizados

- Los límites de inicio de sesión ahora son 10 solicitudes por IP/minuto.
- El registro tiene límites separados: 5 solicitudes por IP cada 15 minutos y 2 por email normalizado cada hora. La clave de email/IP es HMAC-SHA256; usa `RATE_LIMIT_KEY_SECRET` y, si no está definido, `APP_KEY`. En producción el proceso falla al arrancar si ambos faltan.
- Los incrementos usan un script Lua Redis `INCR` + expiración en una operación atómica. El mismo Redis configurado para la aplicación comparte contadores entre instancias. Si Redis no puede aplicar el límite, las rutas de autenticación responden 503 (fail closed); el contador no vuelve silenciosamente a memoria.
- El alta de una empresa opcional, usuario, perfil profesional, relaciones y claves de rol se realiza dentro de una transacción. Cualquier error revierte el conjunto. Se valida el rol/perfil antes de abrirla.
- Emails nuevos y actualizados se convierten a minúsculas y se recortan en los controladores/servicio, además de normalizarse en la consulta de existencia. El índice único actual sigue siendo la última defensa ante dos escrituras simultáneas.
- Se conservó el control de aceptación de carga por actualización condicional y comprobación de filas afectadas, ya documentado en la campaña red-team anterior.

## Verificación ejecutada

- `go test ./app/http/middleware -count=1`: aprobado. Incluye límites, cuota por correo, fallo cerrado, claves sin identidad en claro y contador compartido entre dos aplicaciones.
- `go test -race ./app/http/middleware -count=1`: aprobado. El Redis temporal se omite automáticamente si el sandbox bloquea sockets locales.
- Prueba con Redis temporal real: aprobada fuera del sandbox restringido. 256 incrementos concurrentes devolvieron exactamente los contadores únicos 1–256; también se comprobó expiración y reinicio de ventana.
- Paquetes `viewhelpers`, `billing`, `community`, `exports`, `models` y `monitoring`: aprobados.
- El binario compiló correctamente. Los paquetes de servicios, feature y controladores se compilaron con `go test -c`.
- `git diff --check`: limpio.
- La prueba de servicio `TestCreateWithRole_ConcurrentDuplicateEmailIsAtomic` quedó añadida: lanza 16 altas con el mismo email y empresa nueva, y exige un solo usuario, perfil y empresa final. **Se compiló, pero no se ejecutó**, porque no está disponible una base PostgreSQL de prueba aislada. No se apuntó a la base de producción.

## Riesgos abiertos y controles necesarios fuera de la aplicación

1. **DDoS distribuido:** mantener el origen detrás de CDN/WAF (por ejemplo, el proveedor de hosting), bloquear acceso directo al puerto de origen y configurar límites de tasa/conexiones antes de llegar a Go. Las cuotas por IP no detienen una botnet ni saturación L3/L4.
2. **IP real:** el proxy debe sobrescribir `X-Forwarded-For`; BUFALO solo confía en proxies loopback. Confirmar que la topología de producción conecta desde un proxy local. Nunca confiar en el header enviado directamente por clientes.
3. **Bots de registro:** CAPTCHA administrado o verificación por email antes de activar cuentas todavía no están implementados. Un atacante distribuido puede utilizar muchos IPs y direcciones distintas para crear cuentas. El límite de 2 intentos/hora/email reduce abuso por objetivo pero también permite bloquear durante una hora el alta de una víctima que comparta su dirección; debe combinarse con verificación y un flujo de reintento.
4. **PostgreSQL:** ejecutar las suites feature/servicio en staging con una base `*_test`, incluyendo la nueva prueba concurrente y rollback ante fallo inyectado, antes de desplegar.
5. **Operación:** Redis debe estar privado, autenticado, con persistencia/monitorización apropiadas; no exponerlo a Internet. Las claves `bufalo:rate:v1:*` expiran y están namespaced, y la configuración de limpieza no debe vaciar esa base Redis compartida.

## Criterio de salida

El hardening de aplicación está implementado y sus componentes sin dependencia de PostgreSQL pasan. No se debe afirmar resistencia a DDoS ni desplegarlo como protección completa hasta probar las transacciones sobre PostgreSQL de test, verificar el proxy de producción, configurar WAF/CDN y acordar el flujo de verificación de cuentas. La instancia de producción no fue modificada por esta tarea.
