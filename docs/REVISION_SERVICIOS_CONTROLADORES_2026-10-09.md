# Revisión funcional de servicios y controladores

## Dictamen y alcance

El proyecto es parcialmente funcional. Login y registro tienen defensas sólidas y pasan sus pruebas; no aprobaría el conjunto para manejar información privada y operaciones comerciales sin corregir los hallazgos de prioridad alta. Los límites de transporte y el análisis de dependencias no demuestran que las reglas de negocio sean correctas.

Se revisaron primero UserService, EmailService, EmailVerificationService, limpieza de pendientes, CargaService, FacturaService, EmpresaService, DireccionService, ChoferService, PublicadorService y DriverCommunityService; después AuthController y los controladores de usuarios, cargas, facturas, empresas, direcciones, choferes, publicadores, comunidad, interés y administración, junto con rutas y plantillas relacionadas. Es una revisión enfocada, no una certificación exhaustiva de cada combinación de estado y permiso.

No se modificó código de ejecución ni se desplegaron cambios durante esta revisión. Las reproducciones usaron una superposición temporal de Go y la base aislada `bufalo_auth_utf8_test`, sin enviar correos reales.

## Verificación realizada

Se ejecutó nuevamente `go test -p 1 -count=1` sobre cuatro paquetes: servicios, controladores, feature y redteam. Todos terminaron con código 0: 2,403 s, 10,228 s, 9,753 s y 59,899 s respectivamente, sin contar compilación. Registro: `storage/implementation-backup/code-review-tests-20261009.log`.

Se añadieron cinco reproducciones temporales, ejecutadas mediante `go test -overlay ... -run TestAuditReviewFindings -v ./tests/redteam`. Todas confirmaron el comportamiento defectuoso esperado. Que estas reproducciones pasen significa que el fallo existe; no que esté corregido. Registro: `storage/implementation-backup/code-review-probes-20261009.log`; fuentes de reproducción preservadas en ese directorio con prefijo `code-review-20261009`.

## Evaluación por componente

| Componente | Evaluación | Motivo |
|---|---|---|
| Login | Bien en el flujo probado; mejorable en ciclo de credenciales | Normaliza email, regenera sesión, respuesta genérica, CSRF y cuotas probados. La cuenta se consulta en cada petición para revocar desactivaciones/cambios de rol. |
| Registro y confirmación | Bien en atomicidad y protección del token | Payload cifrado, token aleatorio cuyo digest se almacena, caducidad comprobada tras bloqueo, confirmación de un solo uso y creación transaccional de usuario/empresa/perfil. |
| UserService y perfil | Funcional con riesgo alto en cambios sensibles | CRUD genérico, cambio de correo directo, sin reautenticación ni revocación por cambio de contraseña. |
| Correo | Transporte razonablemente acotado; integración irregular | Plazos y TLS probados. El envío de confirmación es síncrono y ocupa una de dos plazas de registro; el flujo de interés tiene errores de contrato y confianza. |
| CargaService/CargaController | Necesita correcciones prioritarias | Aceptación y tránsito usan actualizaciones condicionales; listado de inicio, direcciones y edición de cargas finalizadas tienen huecos. |
| Facturas | Mejor encapsulado que el CRUD general | Validación de importes/transiciones en servicio, edición condicionada al borrador, control de filas afectadas y cancelación preservando el registro. Falta ampliar cobertura concurrente y consistencia con cambios de la carga. |
| Empresas/direcciones | Propiedad de escritura generalmente comprobada; inconsistente entre accesos | Controladores normales verifican propietario y direcciones, pero listados no están acotados uniformemente y administración duplica validación de forma incompleta. |
| Choferes/publicadores | CRUD funcional; permisos de lectura y datos necesitan definición | La escritura comprueba propietario, pero el detalle de un perfil se entrega a cualquier autenticado; contiene datos de licencia/seguro. Debe definirse la vista pública mínima. |
| Comunidad/calificaciones | Transacciones útiles; concurrencia pendiente de reforzar | Una calificación por carga está respaldada por índice único. El agregado por chofer no serializa calificaciones de cargas distintas. |
| Administración | Funcional en escenarios probados; calidad irregular | Métodos extensos, validación duplicada, errores ignorados y paginación sin tope en varios listados. |

## Hallazgos demostrados

### R1 — Alta: información de cargas privadas expuesta en inicio

`app/http/controllers/AuthController.go:93` consulta cualquier carga publicada, sin aplicar audiencia, pertenencia a red ni propietario. `app/views/home.html:219` renderiza referencia, ruta, recogida, peso, equipo y tarifa. La política de `/loads/:id` sí comprueba visibilidad, por lo que ambas pantallas discrepan.

Reproducción: un chofer ajeno a la red recibe 404 en el detalle de una carga privada, pero ve su referencia en `/home`. No se demostró acceso anónimo; el problema afecta a usuarios autenticados.

### R2 — Alta: paginación permite retirar el límite de filas

`app/services/LoadService.go:84` pasa `perPage` sin normalizar a `Limit`; el controlador lo toma de la query. Con `-1`, el ORM elimina el límite. La misma falta de normalización aparece en servicios de usuarios, empresas, choferes y publicadores. Direcciones, redes y listas del inicio también tienen consultas sin paginación. FacturaService sí limita a 100.

Reproducción: el servicio devuelve todas las cargas sembradas con `perPage=-1`. El controlador de cargas pasa ese valor directamente al servicio. No se provocó agotamiento real de RAM; el riesgo crece con las filas y relaciones precargadas, y las cuotas de solicitudes no limitan el coste de cada consulta.

### R3 — Alta: cargas aceptan direcciones ajenas y cambios tras entrega/facturación

`app/http/controllers/CargaController.go:420` admite los IDs de dirección enviados por el cliente sin comprobar su existencia y propiedad. La lista del formulario muestra solo direcciones permitidas, pero no constituye autorización del POST. `CargaService.Update` actualiza por ID, sin condición sobre el estado.

Reproducción: el propietario de una carga entregada con factura existente cambia su origen a una dirección de otro usuario. Se confirmó que el cambio persiste. Además de exponer relaciones privadas, permite alterar datos comerciales después de finalizar el trabajo. El fallo no permite editar cualquier carga: el control de propietario de la carga sí está presente.

### R4 — Alta: ciclo de cambio de credenciales incompleto

`app/http/controllers/UserController.go:172` cambia directamente el correo; a partir de la línea 190 cambia contraseña sin pedir la actual. SessionAuth comprueba existencia, actividad y rol, pero no una versión de credenciales.

Reproducción: una sesión cambia email y contraseña sin verificar el correo nuevo ni presentar la contraseña actual; una segunda sesión abierta previamente sigue accediendo al perfil. CSRF sigue siendo necesario en el arnés de seguridad. El riesgo es la persistencia y ampliación de acceso de quien ya obtuvo una sesión, no un bypass anónimo de login.

### R5 — Media funcional / alta para abuso del envío: «Me interesa» tiene un contrato incorrecto

`app/views/home.html:265` envía CSRF y `mgs`. `LoadIntersetController.go:39` espera `broker_id`, `status=open` y `msg`. El formulario real termina en «Datos incompletos». Se reprodujo esa respuesta sin enviar correo.

La revisión estática también confirma que, si el cliente aporta los campos faltantes, el controlador confía en destinatario y estado recibidos, no carga la operación ni comprueba su audiencia. Interpola texto no escapado en HTML y usa remitente y URL localhost fijos. No se ejecutó esa rama para evitar mensajes reales; el potencial de notificación manipulada procede de la lectura del código.

## Hallazgos adicionales de revisión estática

- **R6, media:** AdminController.UsersUpdate no comparte las reglas mínimas de contraseña/email del registro. Usa `User.SetPassword` con coste bcrypt por defecto 10, mientras el registro y el hash ficticio de login usan 12; si falla el hash, omite el cambio silenciosamente. EmpresasUpdate, ChoferesUpdate y DireccionesUpdate administrativos también omiten parte de las validaciones presentes en los controladores normales. Unificar política y comprobar errores; no implica acceso administrativo no autorizado.
- **R7, media:** varios GetByID convierten cualquier fallo de base de datos en «no encontrado», y varios controladores ignoran errores de Count/Find. Esto puede devolver 404, listas vacías o éxito aparente ante una caída. ChoferController.Index elimina de hecho el filtro de empresa si no puede resolver el perfil. Debe fallar de forma cerrada y diferenciar errores de infraestructura.
- **R8, media, riesgo concurrente pendiente de reproducción:** DriverCommunityService calcula promedio y cantidad antes de actualizar al chofer, sin bloquear ese chofer antes de leer los agregados. Dos calificaciones de cargas diferentes pueden calcular agregados parciales y sobrescribirlos. El índice único por carga no resuelve ese caso; requiere prueba de concurrencia a nivel PostgreSQL.
- **R9, media/baja:** ShowHome mezcla contadores y listas con los mismos nombres; los contadores administrativos de asignadas/entregadas se sobrescriben con listas vacías. Sus consultas de cargas asignadas/entregadas tampoco tienen límite. Hay además enlaces/acciones inconsistentes, como configuración sin ruta declarada y acciones de chofer mostradas a un admin sin perfil.
- **R10, baja:** logout modifica estado mediante GET y omite el error de Destroy. Usar POST con CSRF y tratar el fallo del almacén. No se afirma que SameSite permita todas las modalidades de logout forzado; se propone eliminar la operación mutante en GET.
- **Regla de producto por definir:** registrar un publicador sin empresa es válido, pero publicar exige un broker activo. Crear luego una empresa desde EmpresaController no enlaza automáticamente user y perfil. Hace falta una ruta explícita de asociación aprobada o una explicación visible del bloqueo. No habilitar asociación arbitraria para resolverlo.

## Plan de mejora y criterios de aceptación

### Fase 1 — Privacidad y coste de lectura

1. Crear una política compartida de visibilidad de cargas para inicio, listado, detalle, aceptación e interés. Comprobar rol, propietario, asignación y red privada dentro de la consulta cuando sea posible. **Aceptación:** un usuario ajeno no recibe referencias ni datos privados en ninguna pantalla; un miembro autorizado conserva acceso.
2. Normalizar page/per_page en todos los servicios: página mínima 1, tamaño predeterminado, tope 100 y tratamiento explícito de negativos, cero, texto y desbordamiento. Paginar direcciones, redes y tablas del inicio; usar selectores con búsqueda para formularios. **Aceptación:** ningún parámetro elimina el límite; consultas y respuesta permanecen acotadas con un conjunto grande de datos.
3. Validar existencia y permiso de ambas direcciones en creación y edición de cargas. Condicionar las modificaciones al estado permitido, con propietario y estado en la misma escritura o transacción. Proteger datos entregados/facturados y separar una corrección administrativa auditada. **Aceptación:** el POST de dirección ajena falla y no cambia filas; dos operaciones concurrentes no eluden la restricción.

Impacto: cambia únicamente el acceso hoy indebido, tamaños de página excesivos y ediciones de operaciones cerradas. Deben conservarse paginación, formularios y flujos legítimos; requiere acordar qué correcciones administrativas se permiten.

### Fase 2 — Identidad y correo

4. Centralizar cambio de credenciales: contraseña actual o reautenticación reciente, confirmación de correo nuevo con token de un solo uso, y versión de sesión/credenciales que permita revocar las demás sesiones. Definir expresamente el procedimiento de un reset administrativo. Unificar bcrypt, longitud mínima y máximo de 72 bytes; actualizar hashes antiguos durante login satisfactorio. **Aceptación:** correo no confirmado no sustituye al vigente y una sesión anterior se rechaza tras revocación.
5. Reparar interés tomando carga, estado, audiencia y publicador de la base; aceptar solo el mensaje del cliente. Escapar HTML, usar APP_URL/remitente configurados y deduplicar por chofer/carga. Aplicar cuota específica de envío y plazos. **Aceptación:** botón real funciona con SMTP capturado; IDs/estado manipulados no cambian el destinatario y un no miembro no notifica cargas privadas.
6. Añadir reenvío de confirmación con cuota y rotación controlada del token. Si se introduce entrega asíncrona, usar una outbox persistente con capacidad, reintentos y retención limitados; no una cola ilimitada en RAM ni un servicio adicional obligatorio. **Aceptación:** caída SMTP no deja una solicitud irrecuperable ni duplica cuentas o correos indefinidamente.

Impacto: el usuario verá un paso adicional al cambiar email/contraseña y un mecanismo explícito para invalidar sesiones. El registro y el login habituales conservan sus rutas; no relajar la aprobación de empresas.

### Fase 3 — Servicios y controladores coherentes

7. Mover reglas de negocio a servicios con actor, entradas tipadas y listas explícitas de campos editables. Compartir validadores entre panel y flujo normal. Mantener controladores como adaptación HTTP; dividir AdminController por recurso sin cambiar URLs. **Aceptación:** la misma entrada inválida falla desde ambos accesos y también al llamar al servicio.
8. Introducir errores distinguibles: no encontrado, prohibido, conflicto, validación e infraestructura. Preservar la causa y comprobar filas afectadas. **Aceptación:** una caída de PostgreSQL devuelve indisponibilidad, no un 404 engañoso ni confirmación de guardado.
9. Serializar calificaciones por chofer antes de insertar/calcular agregados; hacer idempotentes operaciones repetibles. Ampliar pruebas PostgreSQL concurrentes para calificaciones, edición/asignación, facturación y cambios sensibles. Revisar índices/FK/CHECK con un inventario de datos existentes; migraciones aditivas, sin borrado automático. **Aceptación:** agregados coinciden con las calificaciones persistidas bajo concurrencia.
10. Definir campos públicos mínimos de perfiles; restringir licencia/seguro y otros datos según autorización. Corregir contadores/enlaces, POST logout y asociación de empresas. **Aceptación:** matriz de roles con casos permitidos y denegados, incluyendo perfiles ausentes e inactivos.

### Fase 4 — Validación y publicación

Convertir estas reproducciones en regresiones que esperen el comportamiento corregido. Probar UI real, servicios y HTTP con CSRF/sesiones activados; los tests que desactivan límites no sustituyen pruebas del transporte real. Repetir suites, pruebas concurrentes de base de datos, `go vet`, análisis de vulnerabilidades y carga local moderada, midiendo RSS/heap, consultas y latencia.

Publicar por fases con respaldo, candidato de producción, SHA-256 y comprobaciones antes/después en el puerto 3000. Mantener cloudflared y límites de memoria existentes. No prometer funcionalidad intacta solo porque compila: cada fase debe preservar los flujos permitidos y bloquear las reproducciones antes de desplegarse. No hace falta reescribir el proyecto ni añadir microservicios para estas correcciones.
