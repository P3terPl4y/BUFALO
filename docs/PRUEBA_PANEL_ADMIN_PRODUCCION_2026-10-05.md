# Prueba HTTP del panel de administración

**Fecha:** 5 de octubre de 2026  
**Instancia:** `http://127.0.0.1:3000` (proceso de producción local)  
**Método:** solicitudes HTTP con sesión, cookies Secure, CSRF y formularios del sitio. No hubo escrituras directas a la BD.

## Resultado por recorrido

| Recorrido | Resultado |
|---|---|
| Inicio de sesión admin | Correcto: `POST /login` respondió 303 a `/home`; sesión llegó al panel. |
| Dashboard, usuarios, empresas, choferes, publicadores, cargas, direcciones, facturas, métricas y salud | Todas las vistas respondieron 200. |
| Formularios de alta de usuario, carga, empresa, dirección y factura | Todos respondieron 200 al abrirse. |
| Vistas de edición/detalle con registros existentes | Empresa 15, chofer 4, publicador 1, dirección 10, carga 10, usuario 21 y factura 7 respondieron 200. La carga 10 también abrió el detalle admin (200). |
| Exportación de métricas | CSV, XLSX y PDF respondieron 200 con sus tipos MIME correspondientes. `/admin/health` respondió JSON 200. |
| Acceso anónimo a `/admin/users` | Rechazado con 303 a `/login`. |
| Formularios de activar/eliminar usuario | El HTML incluye token CSRF; el botón eliminar apunta a `/admin/users/:id/delete`. |
| Mutaciones sin token CSRF | Toggle y eliminación de usuario, eliminación de empresa/carga/dirección/factura: todas respondieron 403; ninguna solicitud llegó al controlador. Inicialmente el toggle dio 500 por el manejador global, aunque se rechazó. Corregí el mapeo de errores CSRF a 403 y desplegué; el segundo intento y las demás rutas devolvieron 403. Health check quedó en 200. |

No ejecuté eliminaciones, pagos ni altas válidas. Sí hice un ciclo reversible sobre la cuenta de demostración de Miguel Ángel Álvarez (ID 21): la desactivé/reactivé y cambié/restauré su nombre desde los formularios admin; las cuatro solicitudes devolvieron 303 y confirmé en las vistas que los valores originales quedaron restaurados. También envié dos intentos inválidos de alta de usuario y solicitudes de mutación sin CSRF; estas fueron rechazadas antes de mutar registros. No hubo SQL ni acceso directo a la BD.

## Hallazgo corregido: formulario de alta de usuarios

El formulario real en `app/views/admin/users/create.html` solo presenta los campos `_csrf`, `name`, `email`, `password` y `role`. Sin embargo, `AdminController.UsersStore` (`app/http/controllers/AdminController.go`) exige además `city`, `state`, `country` y `radius`, junto con los campos profesionales según rol:

- Publicador: `publicador_numero_licencia_broker` y `publicador_anios_experiencia`.
- Chofer: `chofer_numero_licencia`, `chofer_tipo_licencia` y `chofer_anios_experiencia`.

Envié dos pruebas desde la forma real, una por rol. Ambas recibieron HTTP 200 al re-renderizar el formulario con errores; las búsquedas posteriores no encontraron usuarios creados. La causa quedó confirmada en el código: faltaban controles para los campos obligatorios del servidor y no había sección para asociar empresa.

La corrección local ya alinea la vista con `UsersStore`: agregó ubicación y radio operativo, campos obligatorios de licencia y experiencia para cada rol, y asociación opcional a empresa existente o nueva. El selector restringe opciones al tipo correspondiente al rol; el servidor valida el modo, exige nombre legal al crear empresa, exige una empresa activa del tipo correcto al asociar una existente, conserva los valores al volver con errores y persiste los datos opcionales recibidos de empresa/perfil. También se dejó de ignorar un error de base de datos al comprobar si el correo ya existe.

Se añadieron pruebas feature para la presencia de campos requeridos, rechazo de perfiles incompletos y rechazo de un modo de empresa inválido. También se amplió la prueba de renderizado de plantillas para comprobar la vista con las dos listas de empresas y un rol seleccionado. Pasaron `go build`, `go test ./app/viewhelpers ./app/http/middleware ./app/billing ./app/community ./app/exports ./app/models ./app/monitoring` y la compilación de `tests/feature`. No se pudieron ejecutar los tests feature que escriben en BD: PostgreSQL de test no está disponible (`127.0.0.1:5432` rechazó/bloqueó la conexión). No se intentó usar la BD de producción para esos tests. Por lo tanto, el alta válida de ambos roles aún debe probarse con una BD de pruebas aislada antes de considerarla verificada integralmente.

La instancia que ya escucha en el puerto 3000 respondió HTTP 200 en `/healthz` durante esta revisión. Este binario todavía no contiene la corrección local del formulario; no se desplegó esta corrección ni se escribieron altas nuevas en producción.

## Dictamen actualizado

El defecto del formulario está corregido en el árbol de trabajo, pero el flujo de alta válido no está certificado hasta ejecutar la suite sobre una BD de test aislada. La compilación sí pasó; la conexión de test falló por restricción del entorno. No hubo escrituras directas a la BD ni altas válidas en producción durante esta corrección. No considero el panel completamente validado hasta completar las pruebas de integración aisladas.
