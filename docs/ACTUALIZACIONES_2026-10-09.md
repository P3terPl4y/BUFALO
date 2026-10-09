# Correcciones funcionales y retirada de producción — 9 de octubre de 2026

## Estado del despliegue

La instancia de producción fue detenida y deshabilitada a petición del usuario. También se detuvieron y deshabilitaron sus dos servicios privados Redis. Las tres definiciones instaladas se archivaron en `storage/implementation-backup/removed-production-units` y se retiraron del gestor de servicios (`is-enabled: not-found`). Se conservan la base de datos, los archivos subidos, las copias y los binarios anteriores. No se ha vuelto a desplegar en 3000/3001. El túnel compartido cloudflared no se modificó.

El candidato corregido se verifica solamente en `127.0.0.1:33333`, con PostgreSQL y Redis de pruebas. El SMTP de esa instancia captura mensajes en localhost; no entrega correo real. Esta instancia no es producción.

## Cambios realizados

| Hallazgo de la revisión | Corrección |
|---|---|
| R1: cargas privadas en inicio | Inicio y listados usan filtros por propietario, chofer asignado y pertenencia a red. Un perfil ausente falla de forma cerrada. |
| R2: consultas sin límite | Paginación normalizada en los servicios, máximo 100 filas. Inicio limitado a 20 por sección. Direcciones, redes y conductores de empresa admiten páginas; los formularios de direcciones permiten cargar más opciones y conservan la ruta actual al editar. |
| R3: direcciones ajenas y modificación posterior a entrega | Creación y edición comprueban propiedad de las direcciones en una transacción. La edición y el borrado rechazan cargas asignadas, finalizadas o facturadas. La edición bloquea la fila frente a una aceptación concurrente. Se retiró el método genérico de edición sin actor. |
| R4: cambio inseguro de credenciales | Contraseña actual obligatoria para cambiar contraseña o correo. El nuevo correo solo se activa mediante confirmación de un solo uso. Las sesiones están ligadas al hash de contraseña y correo vigentes. Se conservan los campos del perfil omitidos en un cambio de credenciales. |
| R5: «Me interesa» | Modal con comentario obligatorio. El servidor obtiene carga, estado y destinatario desde la base, comprueba audiencia y envía al publicador real. HTML escapado, remitente y enlace configurados. Deduplicación por chofer/carga, máximo 10 notificaciones por 24 horas y cuota HTTP por cuenta. Si SMTP falla, se libera la reserva para reintentar. |
| R6: validación administrativa | Validadores de actualización compartidos para empresas, choferes, publicadores y direcciones. Contraseñas de 8 a 72 bytes, bcrypt coste 12 y comprobación de errores; hashes antiguos se actualizan tras un login válido. |
| R7: errores ocultos | Los servicios de consulta conservan errores de infraestructura y distinguen ausencia. Los listados y contadores corregidos rechazan indisponibilidad. El listado de choferes no pierde su filtro si falta el perfil. |
| R8: calificaciones concurrentes | Bloqueo del chofer antes de insertar y recalcular promedio/cantidad, manteniendo la unicidad por carga. |
| R9: inicio y acciones | Contadores administrativos separados de las listas; consultas acotadas, enlace de configuración corregido y acciones de chofer restringidas a ese rol. |
| R10: logout mediante GET | GET muestra confirmación; POST con CSRF destruye la sesión y comprueba el error del almacén. |

Las vistas de perfiles ajenos ocultan números y fechas de licencia/seguro a usuarios sin permiso. No se modificó el servicio SMTP para resolver la prueba: el capturador local tenía una respuesta de autenticación incompleta y se corrigió el arnés.

## Modal y protección del envío

El botón abre un diálogo nativo, enfoca el comentario y permite cancelar con Escape, devolviendo el foco al botón. Funciona en inicio y detalle de carga. El comentario tiene un máximo de 1000 caracteres. Los campos de destinatario o estado enviados por un cliente no participan en la elección del publicador.

Además de la cuota persistente de 10 notificaciones por 24 horas, la ruta admite como máximo 10 intentos por hora y dos envíos simultáneos por proceso. El perfil admite 10 actualizaciones por 10 minutos y dos simultáneas por proceso. En producción los contadores HTTP usan Redis; las reservas de interés usan PostgreSQL. El transporte SMTP conserva sus plazos existentes.

Se añadió la migración aditiva `20261009000001_create_load_interests`, con unicidad chofer/carga e índice para la cuota diaria. Solo se aplicó en las bases aisladas. Un futuro despliegue deberá ejecutar las migraciones antes de servir tráfico.

## Validación

- `go test -json -p 1 -count=1 ./...`: **200 resultados aprobados**, incluidos subtests; 127 pruebas de nivel superior y 18 paquetes con pruebas. Cero resultados fallidos.
- Servicios: 2.273 s; controladores: 17.607 s; feature: 30.951 s; redteam: 73.868 s. Los tiempos no incluyen toda la compilación.
- `go vet ./...` y `git diff --check`: código 0, sin diagnósticos.
- Firefox: inicio y detalle, foco/Escape, ancho móvil de 390 px, comentario vacío, envío, duplicado y logout/CSRF aprobados. Dos cargas distintas generaron exactamente dos mensajes capturados; el envío duplicado no añadió correo.
- Manipulación HTTP: `broker_id` apuntando al chofer y `status=closed` no cambiaron el publicador elegido ni sustituyeron la comprobación del estado real. Se comprobó el destinatario de ambos correos capturados.
- `govulncheck ./...`, análisis de símbolos de fuente: aprobado, salida 0, sin vulnerabilidades alcanzables reportadas. El analizador informa un aviso en un módulo requerido cuyo código vulnerable no aparece llamado; no equivale a cero avisos en todas las dependencias. Pico **1.40 GiB**, límite 2 GiB, duración 83.171 s y 81.682 s de CPU. Swap deshabilitada y CPU acotada.
- Compilación Go 1.26.8, `CGO_ENABLED=0`, `-trimpath`, `-ldflags='-s -w'`: 56.25 MiB. SHA-256: `bef789bef7968b32c2156834c9b757b9869631abed0ad059118c8bf0febd41d6`.
- Instancia local tras las pruebas: RSS observado **51.04 MiB**. Es una muestra de esta base y carga local, no un límite garantizado de memoria. Salud y disponibilidad: HTTP 200. Puertos 3000/3001 sin escucha; tres unidades de producción inactivas y retiradas del gestor de servicios.
- `go test -race -p 1` enfocado en validación, credenciales, calificaciones concurrentes y reserva/reintento de notificaciones: aprobado en los tres paquetes; servicios 1,385 s, feature 76,336 s y redteam 109,684 s. Sin carreras detectadas en los casos ejecutados.


El arnés de feature desactiva las cuotas HTTP para aislar reglas de negocio; no sustituye las pruebas con CSRF, cuotas y transporte reales. Se comprobaron CSRF y logout en Firefox contra la instancia local completa. Se corrigió el reinicio de pruebas para limpiar también redes y calificaciones, que antes sobrevivían entre ejecuciones. Dos expectativas antiguas se ajustaron a la política segura: un chofer sin perfil no accede al inicio y una carga completada no se borra.

## Impacto y límites

Los usuarios tendrán que iniciar sesión nuevamente al adoptar esta versión porque las sesiones anteriores no contienen la huella de credenciales. También se revocan sesiones tras cambio de contraseña o confirmación del correo. El cambio de correo y contraseña se realiza por separado, con una explicación en el formulario. Las operaciones comerciales cerradas dejan de admitir cambios genéricos; una corrección administrativa excepcional necesita un flujo explícito y auditado.

La reserva de interés se confirma antes de SMTP para impedir envíos concurrentes duplicados. Una caída del proceso entre la reserva y el envío puede dejar una notificación sin enviar; no existe todavía una outbox transaccional con reintentos. No se promete entrega exactamente una vez.

El plan de mantenimiento sigue incluyendo dividir AdminController por recursos, el reenvío controlado de confirmaciones, ampliar el inventario de restricciones FK/CHECK y definir un flujo aprobado de asociación a empresas. Esos trabajos no se presentan como implementados. La revisión original sigue siendo el registro de los hallazgos anteriores a estas correcciones.

Las pruebas locales y el análisis de dependencias no certifican ausencia total de fallos ni resistencia a millones de solicitudes por segundo. No se hicieron ataques contra servicios públicos ni una nueva prueba masiva de carga en esta actualización.

## Evidencia local

- `storage/implementation-backup/fixes-acceptance-tests.jsonl`: aceptación completa con resultados por prueba.
- `storage/implementation-backup/fixes-final-race.log`: pruebas enfocadas con detector de carreras.
- `storage/implementation-backup/fixes-final-vet.log`: análisis estático.
- `storage/implementation-backup/modal-browser.json`: comprobaciones del navegador.
- `storage/implementation-backup/modal-smtp-validation.json`: destinatario, comentario y escape HTML.
- `storage/implementation-backup/fixes-final-vulnerabilities.json`: análisis acotado y consumo de govulncheck.
- `storage/implementation-backup/fixes-final-runtime.json`: identidad del binario, memoria y estado final de producción.
