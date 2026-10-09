# Informe de prueba de aceptación de BUFALO

**Fecha:** 5 de octubre de 2026  
**Aplicación:** instancia HTTP disponible en `127.0.0.1:3000`  
**Base de datos observada:** PostgreSQL `bufalo` (usuario de aplicación `adminbufalo`)  
**Método:** formularios web HTTP, sesiones independientes, CSRF y solicitudes concurrentes. No se hicieron escrituras SQL directas.

## Alcance y límites

La campaña cubrió los recorridos funcionales descritos aquí, roles de publicador/chofer/admin, aislamiento básico, concurrencia, facturación y vistas de métricas. No es correcto afirmar que se probó cada función de la plataforma sin excepción: no hay navegador gráfico/Playwright disponible en este entorno y no se ejecutó la suite Go completa contra una base aislada `_test`. Tampoco se publicó ni se desplegó un binario nuevo durante esta campaña.

La instancia respondió por HTTP en el puerto 3000 y se usó el formulario web. Las operaciones se enviaron con un cliente HTTP que conserva cookies `Secure` para poder probar el listener local HTTP. Esto ejercita rutas, middleware, controladores, sesiones y formularios, pero no equivale a personas operando un navegador real.

## Datos persistentes de demostración

La lectura inicial de la base indicó 3 usuarios existentes (incluido el admin), 0 empresas, 0 cargas y 0 facturas. Se añadieron mediante los formularios de registro, direcciones, cargas y facturas cuatro cuentas españolas de prueba:

| Nombre | Rol | Correo |
|---|---|---|
| María Fernanda Valdés | Publicadora | `bufalo.demo.maria.20261005@example.com` |
| Raúl Ernesto Hernández | Publicador | `bufalo.demo.raul.20261005@example.com` |
| Daniela Isabel Pérez | Chofer | `bufalo.demo.daniela.20261005@example.com` |
| Miguel Ángel Álvarez | Chofer | `bufalo.demo.miguel.20261005@example.com` |

El alta creó las empresas/perfiles correspondientes. Se registraron seis direcciones con coordenadas; se editó una y se comprobaron accesos ajenos. Se dejó a Daniela en la red privada de María. Los registros se conservan para demostración; no se borraron directamente ni mediante SQL.

## Recorridos y resultados

| Flujo | Resultado observado |
|---|---|
| Registro e inicio de sesión de los cuatro roles de prueba | Correcto; las sesiones llegaron al panel. El admin `admin@example.com` también entró al panel. |
| Dirección: alta, edición, coordenadas inválidas y propiedad | Alta/edición correctas; latitud fuera de rango rechazada; usuarios ajenos no pudieron abrir la dirección de María (404). |
| Red de choferes | Se añadió a Daniela a la red de María; la carga privada de María se mostró a Daniela y no a Miguel. |
| Cargas públicas | María y Raúl publicaron cargas visibles para los choferes. La solicitud de Miguel por una carga privada fue rechazada. |
| Ciclo de carga | Se aceptaron cargas, pasaron a tránsito y se marcaron entregadas mediante formularios. |
| Concurrencia de aceptación | Dos sesiones de chofer enviaron aceptación concurrente para la misma carga. Hubo un único ganador y la carga no se asignó dos veces; duración observada aproximada: 227 ms. La carga quedó completada por el chofer ganador. |
| Facturas | Se crearon tres facturas; se editaron importes, se emitió una y se pagó otra. El pago de una factura en borrador fue rechazado y su estado no cambió. |
| Aislamiento de facturas | La lista varió por actor y participación en la carga. Un usuario ajeno recibió 403 al solicitar una factura privada. |
| Exportaciones | CSV, XLSX y PDF devolvieron 200; se comprobaron BOM/cabeceras CSV, firma ZIP de XLSX y firma `%PDF-`. |
| Métricas y salud admin | Las páginas `/admin/metrics` y `/admin/health` respondieron y renderizaron bajo sesión admin. No se comparó cada contador con consultas SQL de verificación final. |
| Pantallas por rol | Los cuatro usuarios de prueba pudieron abrir las páginas principales y mantener CSRF/sesión. Admin recibió 403 al intentar abrir el recurso exclusivo de red de publicadores. |
| CRUD admin temporal | Una cuenta, empresa, dirección y carga desechables se crearon por formularios web y se eliminaron desde formularios admin. La cuenta no apareció en la tabla y no pudo iniciar sesión; la empresa dejó de aparecer en el índice; la dirección devolvió 404; la carga redirigió al índice con “no encontrada”. Una factura temporal se “eliminó” desde admin, pero la regla la conserva como `cancelada` y su detalle sigue accesible al admin. Las cuentas, empresas, cargas y facturas principales de demostración permanecen. |

Los intentos directos de leer una carga privada devolvieron una página de error con HTTP 200 en vez de HTTP 404/403. No se observó exposición del contenido de la carga, pero el código de estado confunde clientes, monitoreo y pruebas automatizadas.

## Prueba final del panel de administración

Se recorrieron las pantallas de usuarios, empresas, cargas, direcciones, facturas, métricas y salud. Se editó y restauró el nombre de un usuario de demostración, se desactivó/reactivó usuario y empresa para recuperar sus estados originales, y se editó/restauró una dirección y el nombre legal de una empresa. No se borraron registros demostrativos principales. Los paneles de empresa y dirección muestran acciones de edición/eliminación; facturas y cargas exponen acciones limitadas por su estado.

Se intentó crear desde `/admin/users/create` usando el rol `broker` que ofrece la propia pantalla. El servidor respondió `Solo se pueden crear usuarios con rol publicador o chofer` y volvió a mostrar el formulario. **El formulario y el controlador discrepan**, por lo que la creación de usuarios anunciada por el panel falla para ambos roles que presenta (`broker`/`carrier`).

En la comprobación final se reprodujo el `Forbidden` del botón de estado: el formulario renderizado de `/admin/users` no enviaba `_csrf`; el POST equivalente del navegador devolvió 403 para el usuario de María. El controlador sí acepta la acción con CSRF válido. Se corrigió la vista para incluir el token en activar/desactivar y eliminar. También se corrigió el formulario de eliminación: apuntaba a `/admin/users/:id` (ruta de edición) en vez de la ruta registrada `/admin/users/:id/delete`. Esta corrección es común a todas las filas de usuarios. La vista corregida está en el workspace, pero aún no se ha desplegado al proceso del puerto 3000.

Al momento de la prueba, el panel no ofrecía altas para empresas, cargas, direcciones ni facturas, ni edición de cargas. Las entidades sin un estado de activación propio (facturas, cargas y direcciones) no deben recibir un botón de activar/desactivar artificial; sus estados de negocio son distintos. La eliminación admin de una factura es una transición a `cancelada`, conservando historial, no un borrado físico.

## Pruebas automáticas y del cambio de aceptación

Los paquetes Go independientes de la base de datos pasaron (`app/billing`, `app/community`, `app/exports`, `app/http/middleware`, `app/models`, `app/monitoring`, `app/viewhelpers`). `go build` compiló la aplicación. Se añadió una prueba feature para verificar que las filas admin rendericen CSRF en activar/borrar y apunten a la ruta registrada; el binario de pruebas compila, pero el caso no se ejecutó porque la suite requiere una base `_test` aislada.

`go test ./...` **no pasó**: las suites de controladores, servicios, integración/red-team y feature abortaron porque exigen `APP_ENV=testing` y una base con nombre terminado en `_test`; el entorno actual apuntaba a `bufalo`, y el sandbox no permitió conexión al PostgreSQL al usar la configuración predeterminada. La salvaguarda de los tests evitó escribir sobre la base operativa. Hay que volver a ejecutar la suite secuencialmente contra una base aislada configurada correctamente antes de considerar la verificación automatizada completa.

Tras cerrar la prueba principal y el ejercicio admin, se corrigió el botón de aceptar carga: reemplaza el `confirm()` nativo sin estilo por un diálogo accesible y estilizado. El formulario ahora detiene el POST hasta que el usuario pulse “Aceptar”; “Volver”, Escape o el fondo cierran el diálogo sin navegar; el foco queda dentro del diálogo y la confirmación envía una sola vez. Se validó el manejador real con un DOM simulado de Node (pausa, cancelar, Escape, foco y confirmar) y se recompiló el binario. **No se actualizó el listener de producción ni se probó visualmente en navegador real.**

## Correcciones posteriores a la campaña (workspace)

Las siguientes correcciones se compilaron y desplegaron al proceso del puerto 3000 el 5 de octubre de 2026:

1. **Alta de usuario admin:** el formulario envía ahora roles válidos `publicador`/`chofer` y conserva etiquetas funcionales Broker/Carrier. Se añadió cobertura feature del HTML renderizado.
2. **Activación y eliminación de usuarios:** ambas acciones incluyen CSRF; borrar apunta a `/admin/users/:id/delete`, que es la ruta registrada. Se añadió prueba feature de acciones de cada fila.
3. **Alta y edición desde Admin:** accesos a alta de empresa/dirección/factura, alta y edición de carga y edición de factura conectan con rutas protegidas. Para alta de carga admin se exige un publicador existente ligado a empresa broker activa; la empresa de la carga se toma del perfil y no de un ID arbitrario enviado por el formulario. Las acciones operativas de asignar chofer y calificar siguen reservadas al publicador.
4. **Códigos de respuesta:** los detalles inexistentes de carga, factura, empresa, chofer y publicador devuelven 404 real. El manejador HTTP global ahora presenta una página visual consistente para 400, 401, 403, 404, 405, 408, 409, 413, 419, 422, 429, 500, 502, 503 y 504; la ilustración SVG tiene movimiento CSS y respeta `prefers-reduced-motion`. Los clientes que solicitan JSON conservan respuestas JSON, y el cliente nunca recibe errores internos.
5. **Aceptar carga:** se implementó el diálogo con estilo, teclado y control de doble envío documentado arriba.

## Validación de las correcciones

`go build` terminó correctamente. Pasaron `go test` para middleware (incluida prueba HTTP del render 404 con estado real), billing, community, exports, models, monitoring y viewhelpers. El paquete feature compila con `go test -c`; los casos feature no se ejecutaron. La suite completa se intentó exclusivamente con `APP_ENV=testing` y `DB_DATABASE=bufalo_validation_test` en `127.0.0.1:55441`, pero la conexión fue rechazada. Por tanto, no hay evidencia automatizada aún de los cambios de controlador contra base aislada, ni prueba visual de navegador para el diálogo/páginas.

Después del despliegue final, `/healthz` y `/login` devolvieron HTTP 200, la ilustración SVG HTTP 200 y una URL desconocida HTTP 404 con el título BUFALO “Página no encontrada”. La primera sustitución no inició y dejó el puerto sin listener; recuperé el binario anterior, revisé la sesión y volví a aplicar la versión con arranque registrado. La segunda comprobación confirmó el nuevo servicio arriba. No se ejecutó migración nueva ni se modificaron registros durante el despliegue.

El hash SHA-256 del ejecutable activo coincide con el binario compilado: `6da974881277e65ccb33b074c052db490d8d4a3d3b2523052d6355e8180a8716`. Se conservaron copias previas en `/tmp/bufalo-release.backup-20261005` y `/tmp/bufalo-release.backup-before-404-fix`. No se ejecutó migración nueva ni se modificaron registros durante el despliegue.

## Cola priorizada pendiente

1. **Crítica — Suite aislada:** levantar o provisionar `bufalo_validation_test` en el puerto de test documentado y ejecutar la suite completa; no apuntar pruebas a `bufalo`.
2. **Alta — Pruebas funcionales de correcciones:** cubrir alta/edición admin de cargas e invoice, usuarios, errores HTTP, autorización y ausencia de datos privados.
3. **Alta — Verificación en navegador:** ejercitar diálogo de aceptación y páginas de estado en Chromium (móvil, teclado, JS, carga lenta).
4. **Media — Métricas:** cotejar contadores del panel con consultas `SELECT` de solo lectura y probar emisores de factura sin empresa.
5. **Despliegue:** completado y smoke test básico confirmado; falta probar rutas autenticadas de admin y los demás estados HTTP antes de declarar versión estable. Sin push realizado.

## Dictamen

BUFALO completó recorridos manuales representativos de registro, red, direcciones, cargas, concurrencia, facturación y exportación. Las correcciones quedaron desplegadas en el puerto 3000 y el smoke check básico está verde. **Aún no recomiendo declarar la versión estable/lista para producción**: la suite de integración necesita una base aislada que no está disponible y faltan pruebas de navegador y humo autenticado de nuevas rutas admin. No se hizo push.
