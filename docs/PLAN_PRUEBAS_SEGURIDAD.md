# Prueba adversarial y plan de mejora de BUFALO

Fecha de ejecución: 2 de octubre de 2026. Entorno: PostgreSQL aislado `bufalo_redteam_test`, sesiones en memoria, HTTP mediante Fiber y cuentas/datos sintéticos. La aplicación que escucha en el puerto 3000 y su base de datos no participaron en estas pruebas.

## Resultado de la ronda de 100 cuentas

Se crearon y autenticaron exactamente 100 usuarios: 34 publicadores, 33 choferes y 33 administradores. Cada cuenta tuvo su propio cliente HTTP, cookie y token CSRF. El arnés ejercitó 800 peticiones adversariales concurrentes (20 trabajadores): página autenticada, lectura de una factura perteneciente a otra empresa, acceso a administración, intento de pagar factura ajena o en borrador, modificación de dirección ajena, aceptación concurrente de una carga, envío de factura con campos manipulados y escritura desde origen externo con token CSRF válido. Se añadieron controles anónimos, token CSRF ausente, perfil profesional ausente y comprobación del estado final de la carga.

Resultado de `go test -v ./tests/redteam -count=1 -timeout=180s`: aprobado. No hubo discrepancias entre respuestas esperadas y observadas. La única carga publicada disputada acabó asignada a un chofer, sin transición inválida; la operación de aceptación usa una actualización condicionada por estado y `chofer_id IS NULL`. La prueba demuestra este caso de concurrencia con PostgreSQL, no la ausencia general de carreras en todos los servicios.

## Hallazgos

| ID | Severidad | Evidencia | Acción / riesgo restante |
|---|---|---|---|
| RT-01 | Alta, mitigado | En `CargaController.Index`, si no se encontraba el perfil del usuario, se omitía el filtro `publicador_id` / `chofer_id`; la consulta posterior podía devolver cargas de terceros. | Ahora responde 403 si no se resuelve el perfil. La prueba elimina un perfil de chofer ya autenticado y verifica denegación. No se verificó con datos de producción ni se creó una cuenta rota adicional. |
| RT-02 | Media, no vulnerabilidad observada | Mutaciones de la matriz devuelven redirects tanto en rechazo de rol como en ciertos errores de negocio; esto hace difícil distinguir permiso denegado de fallo de validación con el código HTTP. | Normalizar denegaciones a 403/404 y errores de validación a 400/422; aserciones deben comprobar además estado persistido y ausencia de efectos secundarios. |
| RT-03 | Media, pendiente de alcance | Prueba HTTP usa sesiones en memoria y salta el rate limiter para mantener pruebas funcionales deterministas. | Probar Redis, cookies/TLS, límites por cuenta e IP, fijación y revocación de sesión en entorno de integración separado. Existe test aislado del limiter, pero no de la protección a carga real. |
| RT-04 | Media, pendiente de alcance | El conjunto adversarial cubre perfiles, dirección, carga e invoice; no recorre todas las mutaciones administrativas, empresas, perfiles, registro, logout y recuperación de credenciales. | Completar matriz de rutas y permisos indicada más abajo antes de aceptar la plataforma como verificada exhaustivamente. |
| RT-05 | Media, pendiente | Los errores del controlador se redactan en logs y las respuestas no se validan de forma uniforme por esquema/código; algunas rutas redirigen con un mensaje genérico. | Validación server-side por campo, códigos coherentes, límites de tamaño, contenido y longitud, y errores sin detalles internos. |
| RT-06 | Alta, mitigado en código y regresión aislada aprobada | La asignación manual exige un perfil de chofer existente y disponible y actualiza la carga solo si sigue publicada y sin asignar; la transición también fija `estado=asignada`. | La prueba dirigida verificó ID inexistente, chofer ocupado, asignación válida, estado final y doble asignación; la suite red-team completa también pasó en PostgreSQL efímero. Falta validarlo en una instancia viva con su base y Redis configurados. |

No se observó evasión de CSRF por token ausente ni por `Origin` externo, acceso a `/admin` por los roles no administrativos ni lectura/pago de la factura del broker ajeno. Los importes malformados fueron enviados por los 100 usuarios; las respuestas no revelaron excepción. El arnés comprueba el resultado HTTP, aunque la política fiscal y cada validación de campo se cubren mejor en pruebas unitarias/servicio.

## Plan de prueba exhaustivo

Se añadió `TestProductionJourneys` en `tests/redteam/redteam_test.go` para recorrer CRUD de cargas, direcciones y empresas, aislamiento de propietario, ciclo de facturación y páginas de cada rol, verificando persistencia con CSRF. En esta continuación no se pudo obtener un resultado de ejecución: el proceso de prueba se quedó esperando antes de reportar casos al usar el PostgreSQL aislado. Por tanto, ese código de prueba requiere una ejecución repetida en CI o en el entorno de pruebas con DB accesible. La matriz siguiente sigue pendiente en los aspectos restantes; guardar salida y versión del commit, y no permitir salida a servicios externos.

| Fase | Casos | Criterio de aprobación |
|---|---|---|
| 1. Inventario | Enumerar cada método/ruta, middleware, rol requerido, dato de propiedad y cambio de estado; revisar rutas web, middleware y controladores. | No queda ruta sin dueño funcional, autorización ni prueba asociada. |
| 2. Identidad y sesión | Login correcto/incorrecto, usuario desactivado/eliminado, rol cambiado durante sesión, logout, expiración, regeneración de ID, cookie manipulada, intentos rápidos y recuperación si existe. | Sesión revocada al cambiar cuenta, sin fijación ni enumeración de usuarios; límites efectivos en Redis/producción. |
| 3. Autorización por objeto | Para cada rol, probar ID propio, de otra empresa, de otro usuario, inexistente, 0, negativo, overflow, UUID/texto, secuencial y enumeración; probar GET, POST, PUT, PATCH y DELETE disponibles. | Ningún dato privado ni efecto lateral cruza propietario/empresa; 403 o 404 consistente. |
| 4. Cargas | Crear, editar, asignar, aceptar, interesarse, cancelar/borrar y transiciones de estado; dos choferes aceptando simultáneamente, propietario editando durante aceptación, perfil faltante, estados terminales y acceso según audiencia. | Una sola asignación gana; persistencia respeta transiciones y propiedad en BD, no solo en UI. |
| 5. Facturación | Ciclo borrador→emitida→pagada/vencida/cancelada, edición y duplicados; carga no entregada, empresas no relacionadas, usuario de empresa ajena, importe negativo/NaN/Inf/máximo, decimales límite, monedas, fechas y doble pago concurrente. | Total calculado en servidor, transición idempotente/atómica, unicidad y relaciones protegidas por aplicación y base. |
| 6. CSRF y navegador | GET/POST/PUT/DELETE, token ausente, adulterado, caducado, otro usuario, origen y referer externos, SameSite, HTTPS y formularios multipart. | Todo cambio requiere token de la sesión correcta; no hay mutaciones por GET ni bypass por método alternativo. |
| 7. Validación y robustez | Campo omitido/nulo/vacío, Unicode y normalización, HTML/script, SQL metacharacters, JSON malformado, cuerpo grande, paginación extrema, tipos incorrectos y valores límite. | Rechazo controlado, sin stack trace, corrupción, consultas sin límites ni XSS almacenado/reflejado. |
| 8. Concurrencia y resiliencia | Doble submit, edición simultánea, deadlock/timeout, DB/Redis inaccesible, reinicio, worker duplicado y reintentos. | Transacciones atómicas; errores explícitos; no se duplica cobro/factura/asignación; recuperación verificable. |
| 9. Seguridad de dependencias y configuración | `govulncheck`, revisión de dependencias, secretos, modo debug, cabeceras, TLS, CORS, permisos de archivos, backups y logs. | Sin vulnerabilidades conocidas altas sin aceptación documentada; secretos fuera del repo; entorno productivo endurecido. |
| 10. Carga | 100, 250 y 500 sesiones; mezcla realista por rol, tasas sostenidas y picos; lecturas/escrituras y base observada. | Umbrales de latencia/error definidos antes del test, DB estable y límites anti abuso efectivos. |

## Plan de mejora priorizado

1. **P0 — cerrar aislamiento de datos.** Mantener la corrección RT-01 y aplicar una política central por recurso a cargas, direcciones, empresas, choferes/publicadores y facturas. Agregar pruebas negativas cruzadas para cada ruta de lectura y escritura, comprobando filas de BD sin cambios tras denegación.
2. **P0 — unificar sesión y revocación.** Hacer que todas las rutas consulten el estado activo actual y rol vigente; probar sesión Redis real, expiración, rotación de cookie y cambios de rol. El comportamiento de `IsActive` debe ser consistente fuera del módulo de factura.
3. **P0 — transacciones e integridad.** Usar transacciones para alta con perfil/empresa y operaciones de negocio; agregar FK, CHECK e índices únicos mediante migración compatible, tras auditar datos preexistentes. Nunca aplicar `fresh` a datos de usuario.
4. **P1 — facturación segura y auditable.** Sustituir `float64` por decimal/centavos; guardar eventos inmutables de emisión, pago y cancelación; agregar idempotencia; probar doble submit/doble pago con concurrencia SQL. Definir reglas comerciales/fiscales y zonas horarias antes de emitir documentos reales.
5. **P1 — errores previsibles.** Devolver códigos HTTP distinguibles para autenticación, autorización, validación, conflicto de estado y fallo interno. No filtrar errores internos. Registrar actor, recurso, decisión y request ID sin credenciales ni datos sensibles.
6. **P1 — matriz de rutas automatizada.** Convertir la fase 1–4 en tabla de pruebas mantenible que cubra todas las rutas y perfiles, tanto permitido como denegado, con prueba de no mutación.
7. **P2 — operación y rendimiento.** Integrar escaneo de dependencias, análisis estático, pruebas con Redis y carga sostenida en CI; establecer SLO y alarmas, revisar backups/restauración y retención de logs.

## Comandos reproducibles

```sh
APP_ENV=testing DB_CONNECTION=postgres DB_HOST=127.0.0.1 DB_PORT=55441 DB_DATABASE=bufalo_redteam_test DB_USERNAME=peter DB_PASSWORD='' go test -v ./tests/redteam -count=1 -timeout=180s
go test -race ./app/billing ./app/http/middleware
go vet ./...
git diff --check
```

La base de prueba se trunca y debe ser exclusiva. La prueba de 100 usuarios es una prueba adversarial dirigida, no una auditoría de penetración externa ni una demostración de que la plataforma esté libre de vulnerabilidades. La cobertura completa requiere ejecutar las fases pendientes, revisar despliegue/TLS/Redis y definir umbrales de rendimiento y reglas fiscales.

## Hallazgos del despliegue activo en `127.0.0.1:3000`

Ejecución directa contra el proceso existente, no contra Fiber de prueba. La configuración leída sin revelar secretos indica `APP_ENV=local`, `SESSION_DRIVER=file` y base PostgreSQL `bufalo`; por tanto, corresponde al despliegue activo que se pidió probar, aunque no está configurado con `APP_ENV=production`. Se usaron cuentas y registros con prefijo único `LIVEAUDIT-*`, sin ejecutar migraciones ni reiniciar el proceso. Tras la prueba se verificaron usuarios, empresas, perfiles, cargas, facturas y direcciones por marcador exacto; PostgreSQL confirmó cero filas residuales en esas siete comprobaciones.

| ID | Resultado vivo | Impacto y próximo paso |
|---|---|---|
| LIVE-01 | Antes de corregir, `GET /publicadores` devolvía 500 porque faltaba `publicadores/index`. Se añadieron vistas de listado, detalle y edición. | Corregido. Repetición en el proceso actualizado: listado y edición del perfil devuelven 200. |
| LIVE-02 | Antes de corregir, `GET /admin/cargas` devolvía 500 al evaluar `.Publicador.User.Name`; faltaba precargar la relación anidada y la vista no admitía usuario ausente. | Corregido con precarga anidada y guardas en plantilla. Repetición con filas realistas en el proceso actualizado: 200. |
| LIVE-03 | Una cuenta de otra empresa podía listar y abrir por ID una dirección de propiedad ajena. | Corregido: listado y selectores filtran por propietario para cuentas normales; detalle ajeno devuelve 404; admin conserva el listado global. Repetición en el proceso actualizado confirmó aislamiento de detalle y listado. |
| LIVE-04 | La primera prueba registró 403 al emitir, pero su helper omitía `_csrf` cuando el formulario era vacío. El rechazo era CSRF correcto, no error de autorización de emisión. | Corregido el arnés de prueba para incluir el token. Repetición HTTP con token válido completó borrador→emitida→pagada; el total enviado por el cliente se sustituyó por `100 + 10 = 110`. |
| LIVE-05 | La primera prueba registró 403 al aceptar carga por la misma omisión de `_csrf` en formularios vacíos. El rechazo era CSRF correcto. | Repetición HTTP con token válido asignó la carga al chofer autorizado. El rol publicador ajeno conserva el rechazo. |
| LIVE-06 | El registro público con rol `admin` fue rechazado; un registro válido de publicador creó la cuenta y empresa sintéticas. | Control de privilegio correcto en el flujo probado. Falta probar los demás errores de validación y la creación de chofer por HTTP. |
| LIVE-07 | `GET /home`, `/loads`, `/direcciones`, `/empresas`, `/choferes` y `/facturas` respondieron 200; panel admin, usuarios, empresas, choferes, publicadores, direcciones y facturas respondieron 200. Publicador y chofer recibieron 303 al entrar a rutas admin. | Lecturas básicas y aislamiento de administración confirmados en estas rutas; no equivale a cobertura de todos los enlaces/paginadores/filtros. |
| LIVE-08 | POST sin CSRF y POST con origen externo devolvieron 403. Modificar dirección propia funcionó (303); modificar dirección ajena devolvió 403. Logout retiró acceso posterior a `/home`. | Controles comprobados para estas sesiones. |

La primera medición de emisión y aceptación tuvo un defecto en el arnés: al enviar formularios vacíos no incluía el token CSRF y el middleware respondió correctamente 403. Se corrigió el arnés y ambos flujos completaron en el despliegue actualizado con token válido. No se relajó la política de autorización.

Se reconstruyó y reinició BUFALO en el puerto 3000 con los cambios. La comprobación de repetición devolvió `LIVE_RETEST_PASS` y una consulta posterior confirmó cero residuos sintéticos en usuarios, empresas, perfiles, cargas, facturas y direcciones. La configuración sigue siendo `APP_ENV=local`, `SESSION_DRIVER=file`; el middleware de sesión principal usa Redis. La corrección RT-01 para perfil profesional ausente se cubrió en la base aislada, no se provocó sobre cuentas del despliegue.

## Comprobaciones continuadas

La asignación manual quedó limitada a perfiles existentes en estado `disponible` y a cargas `publicada` con `chofer_id IS NULL`; carga y estado se actualizan en una sola operación condicional. La regresión dirigida pasó con casos de ID inexistente, chofer no disponible, éxito, estado final y segunda asignación; `TestProductionJourneys` pasó. También pasaron `go test -p 1 ./... -count=1 -timeout=240s`, `go vet ./...`, `go build -o /tmp/bufalo-app .` y `git diff --check`.

El clúster de datos configurado para la instancia (`bufalo` en PostgreSQL 5432) está detenido y pertenece a `nobody`; el contenedor impide elevar privilegios para iniciarlo. Para resolverlo sin tocar esa base, se desplegó el binario en `127.0.0.1:3000` usando PostgreSQL y Redis efímeros aislados en `/tmp`, con `APP_ENV=local` y cuentas exclusivamente sintéticas. El smoke test HTTP del proceso real pasó registro, inicio/cierre de sesión en admin/publicador/chofer, páginas protegidas por rol, creación de direcciones y cargas, asignación manual, aceptación de chofer y ciclo factura borrador→emitida→pagada; la BD confirmó el estado asignado y el total calculado `110.00`. Se verificó después que el proceso sigue activo y `/login` responde 200. Esta ejecución valida el binario y sus rutas en un despliegue local operativo, pero no permite inspeccionar ni afirmar el estado de los datos de producción. Redis emitió únicamente la advertencia del kernel sobre `vm.overcommit_memory`; los flujos no mostraron errores de aplicación.

## Repetición de flujos por curl y correcciones de octubre 2026

La repetición del 3 de octubre se hizo sobre el servidor HTTP activo en `127.0.0.1:3000`, `APP_ENV=local`, PostgreSQL/Redis temporales y las tres cuentas `stage-*`. Todas las altas, ediciones, asignaciones, transiciones de estado y cambios de factura descritos abajo se enviaron por formularios web con `curl`, cookies de sesión y CSRF. Las consultas SQL fueron `SELECT` únicamente para comprobar persistencia. La suite Go usó aparte `bufalo_flow_test`, base desechable con sufijo `_test`; no se usó esa base para operar el panel.

| Hallazgo | Evidencia | Corrección y comprobación |
|---|---|---|
| Coordenadas GPS redondeadas | El formulario envió `23.113592, -82.366592`, el controlador las recibió como `float64`, pero PostgreSQL las guardó como `23.11, -82.37`. El modelo declaraba escala 7, mientras la migración original creaba `Decimal` con la escala por defecto 2. | Se añadió `20261003000001_fix_direccion_coordinate_precision`: `numeric(10,7)` nullable en latitud/longitud. Se aplicó con `artisan migrate` al staging. Por curl se creó y editó una dirección y se leyó `23.1135920/-82.3665920` y luego `23.1135930/-82.3665930`. Latitud 91 fue rechazada sin persistir, y el mensaje ahora aparece en el formulario. |
| Preview del mapa no se refrescaba al elegir un punto | `setMarker` rellenaba los campos por JavaScript, pero no disparaba `input`; el resumen solo escuchaba eventos del formulario. | El marcador y la geocodificación inversa disparan `input` en los campos afectados para actualizar el resumen. Revisión estática hecha; la ronda usa curl y no ejecuta el JavaScript del navegador. Añadir prueba de navegador para búsqueda/selección, permisos de geolocalización y respuesta del servicio Nominatim. |
| No se podía completar una carga desde la UI del chofer | La web permitía aceptar una carga (estado `asignada`) pero no tenía acciones para `en_transito` ni `entregada`; por eso no era posible crear por el panel una factura que exige carga entregada. | Se añadieron transiciones POST condicionadas por carga, chofer asignado y estado esperado; el detalle muestra “Iniciar tránsito” y luego “Confirmar entrega” solo al chofer asignado. La prueba HTTP recorrió publicar→aceptar→tránsito→entrega, confirmó `fecha_entrega` y creó, editó, emitió y pagó una factura completa mediante formularios curl. |
| El mensaje de error de coordenadas no llegaba a la plantilla de alta | La validación redirigía con `flash_error`, pero `DireccionController.Create` no incluía ese valor al renderizar. | La vista de creación ahora recibe el mensaje. La prueba por curl comprobó que “Latitud inválida” aparece en la página tras enviar coordenada fuera de rango. |
| Dos falsos fallos del arnés al recorrer formularios | El primer helper buscó CSRF en páginas de detalle/listado que no contenían formulario. Otra prueba intentó aceptar una carga ya asignada, que correctamente no ofrecía botón de aceptación. | Se tomó el token de formularios reales y se separaron los casos: asignación por publicador y aceptación de una carga publicada independiente. No eran defectos de autorización de BUFALO. El último fallo de prueba fue una aserción que decodificó `%20` pero no `+`; se corrigió a `unquote_plus` y la denegación del chofer quedó verificada. |

La última ejecución por curl pasó autenticación y páginas de admin/publicador/chofer, restricciones de rol, precisión y límites de coordenadas, creación/edición de direcciones, ciclo de carga completo, cálculo server-side del total, borrador→edición→emisión→pago y las vistas rediseñadas de lista/detalle/creación/edición de facturas. Se conserva en staging la carga `THREEFLOW-DELIVERY-FINAL2-20261003` y la factura pagada `THREEFLOW-INV-FINAL2-20261003` (total `137.50 USD`) para inspección. No se borra esa evidencia sintética.

También se enviaron por curl una transición inválida desde `entregada`, una entrega intentada por el publicador y un pago duplicado; los tres casos conservaron intactos los estados y el importe.

Validación de código: `go test -p 1 ./... -count=1 -timeout=240s` aprobado, incluido `tests/redteam` (100 usuarios sintéticos), `go vet ./...`, `go build -o /tmp/bufalo-app-next .` y `git diff --check`. Se reconstruyó y reinició el staging con esa build; después del reinicio `/login` respondió 200 y se repitieron los flujos web anteriores.

La ejecución no prueba JavaScript en un navegador real, búsqueda/geocodificación externa ni operación contra la base productiva. El staging escucha en 3000 y usa `/tmp/bufalo-live-pg-20261003` con la base `bufalo_stage`; no es la base `bufalo` de PostgreSQL 5432.
