# Informe de pruebas funcionales E2E de BUFALO

**Fecha:** 2026-10-04
**Objetivo:** recorrer los flujos visibles como usuario, con cuentas y datos marcados como QA, y detectar bloqueos funcionales en el servicio del puerto 3000.
**Alcance real:** se recorrieron formularios en `http://127.0.0.1:3000` con cookies y tokens CSRF, sin escribir directamente en la base de datos. Ese listener local usa un despliegue aislado y no se debe asumir que comparte BD con el dominio público. En `https://bufalo.duohnson.com/profile`, una consulta sin sesión redirigió a `/login`; un POST de login desde curl recibió 403 con `Server: cloudflare`, por lo que no se pudo autenticar ni confirmar si las cuentas QA existen en el dominio público. No había navegador gráfico ni Playwright disponible; por eso no se validaron presentación visual, JavaScript del mapa ni interacción táctil.

## Plan teórico ejecutado

1. Verificar el servicio y los formularios públicos de inicio de sesión y registro.
2. Crear cuentas QA de chofer y publicador, comprobar autenticación correcta/incorrecta y rechazo de privilegios autoadministrados.
3. Probar permisos por rol, lectura/edición de empresas y aislamiento entre propietarios.
4. Ejecutar CRUD de direcciones, comprobar coordenadas y usarlas en cargas.
5. Publicar una carga, buscarla como chofer, aceptarla, iniciar tránsito y entregarla.
6. Probar perfil, red privada, puntuación y ordenamiento de choferes.
7. Crear una factura vinculada a una carga entregada; comprobar acceso emisor/receptor, borrador, emisión y pago, incluidos datos inválidos.
8. Verificar salida de sesión, errores de rutas y registrar lo que no se pudo ejecutar.

## Resultados ejecutados

| Flujo | Resultado | Evidencia observada |
|---|---|---|
| Landing, login y registro | Parcialmente correcto | Landing, login y registro cargan. Se registraron tres cuentas QA: dos choferes y un publicador. Las tres autenticaron y llegaron a `/home` (200). Una contraseña incorrecta mostró “Credenciales incorrectas”. |
| Registro con rol admin | Correcto | El formulario rechazó `role=admin` con “Debes seleccionar un tipo de cuenta válido”. |
| Perfil y foto | Fallido en el despliegue observado; mitigado en código local | `/profile` y `/profile/edit` devolvieron 500 en la instancia probada. El error exacto de producción compartido después (`.user.ProfilePhoto` sobre `interface{}`) corresponde al camino de error de `UserController.Show`: falla la segunda consulta `GetByID`, el controlador omite `user` del mapa de la vista y la plantilla lo desreferencia. La razón original de esa segunda consulta aún debe confirmarse con los logs del backend. |
| Catálogo y ordenamiento de choferes | Fallido | `/choferes` devuelve 500 al renderizar `RatingAverage`; no se pudo verificar búsqueda, orden por puntuación ni perfil público. |
| Red de choferes | Fallido | `/red-choferes` devuelve 404 para el publicador, pese a que el código de esta copia registra esa ruta. No se pudieron probar alta/baja de miembros ni audiencia privada desde el panel. |
| Empresas y permisos | Parcialmente correcto | El publicador pudo editar su empresa (200); ambas cuentas recibieron redirect (303) al intentar editar la empresa de la otra cuenta. El publicador y el chofer asociado a empresa pudieron listar empresas. No se completaron actualización ni borrado de empresa. |
| Direcciones | Correcto en CRUD básico | Se crearon direcciones con latitud/longitud; sus detalles mostraron las coordenadas y el formulario de carga las ofreció como origen/destino. Una dirección temporal se creó, actualizó y borró por formularios web. No se verificó el JavaScript del mapa/geocodificador. |
| Publicar carga | Correcto | El publicador creó cargas QA `BUFALO-QA-LOAD-20261004` y `BUFALO-QA-INVOICE-20261004`; los detalles reflejaron rutas y coordenadas. |
| Descubrir carga desde el listado de chofer | Fallido | La carga pública aparece en el listado del publicador, pero no en `/loads` del chofer (`No hay cargas`). El detalle directo sí carga y muestra el botón de aceptación. El flujo habitual de descubrimiento queda bloqueado. |
| Aceptar, tránsito y entrega | Correcto por enlace directo | El chofer aceptó la carga y ejecutó las transiciones `asignada → en tránsito → entregada`; las acciones respondieron 303 y el detalle mostró “Entregada”. |
| Puntuar al chofer | Fallido | El detalle de una carga entregada muestra el formulario de puntuación, pero su POST no está operativo: el envío válido termina en 405 Method Not Allowed (también se observaron 403 durante intentos previos). No se confirmó persistencia de puntuación. |
| Facturación | Correcto en ciclo principal | Se creó la factura QA `BUFALO-QA-INV-20261004` vinculada a la carga entregada y empresas QA. El emisor y la empresa receptora pudieron leerla (200); un chofer sin la empresa receptora recibió 403. El borrador pasó a emitida; un pago sin método fue rechazado con mensaje de error; con método válido pasó a pagada y quedó visible en el detalle. |
| Cierre de sesión | Correcto | `/logout` redirigió a login; la sesión ya no pudo volver a `/home` y recibió redirect a `/login`. |
| Recuperación de contraseña | No disponible | El enlace visible `/forgot-password` termina redirigiendo a `/login`; no se encontró una ruta de recuperación en el código revisado. |
| Rol admin | No probado autenticado | El registro público impide crear admin y no se facilitó una cuenta/credencial QA administrativa. Las rutas `/admin/*` redirigen sin una sesión admin. No se probaron operaciones administrativas ni se intentaron contraseñas predeterminadas. |

## Defectos y prioridad

### P1 — El backend que responde no coincide con el código/plantillas actuales

El error `RatingAverage` sí indica que el tipo `models.Chofer` cargado por el listener local no contiene un campo que está en el modelo de esta copia. Además, el listener local responde 404 a `/red-choferes` y 405 al POST de valoración, aunque esta copia registra ambos endpoints. Esto demuestra una discrepancia entre el servicio local y el código actual; la causa probable es un backend antiguo o un despliegue parcial. Para el dominio público, el propietario reportó el mismo error autenticado; la prueba anónima no permite atribuirle el mismo proceso/binario. El error `ProfilePhoto` sobre `interface{}` tiene otra causa inmediata: la vista recibe el mapa de error sin `user` después de fallar `GetByID`. No se pudo consultar un identificador de build del proceso para determinar qué binario exacto está activo.

En la copia local, `SessionAuth` ya carga el registro del usuario y confirma que está activo. Se cambió el controlador para reutilizar ese registro durante la petición, evitando una segunda consulta con `With("Empresa")`; si la ruta de fallback no logra cargarlo, ahora redirige al login en lugar de renderizar una plantilla sin `user`. El cambio está probado por compilación, análisis estático y pruebas unitarias no dependientes de base de datos, pero **no está desplegado ni verificado en el dominio público**.

La respuesta pública observada en HTTPS emitió cookies `csrf_` y `session_id` sin atributo `Secure`. El código de esta copia solo usa cookies `__Host-*` y `Secure` cuando `APP_ENV=production`; la respuesta apunta a `APP_ENV` ausente/local o a una versión antigua. El POST curl bloqueado por Cloudflare no se considera una prueba concluyente del login en un navegador real.

**Impacto:** bloquea perfil, foto, catálogo/ordenamiento y red de choferes; deja formularios visibles que no tienen una acción funcional en el backend. Debe desplegarse backend, migraciones y plantillas desde el mismo commit y verificarse su versión antes de repetir E2E.

### P1 — Cookies de sesión públicas sin `Secure`

La respuesta HTTPS del dominio público emitió `session_id` y `csrf_` sin el atributo `Secure`, y no usó los nombres `__Host-*`. Esta copia solo activa esos atributos cuando `APP_ENV=production`. Revisar el entorno del servicio público y confirmar después de reiniciarlo que la sesión y CSRF usan cookies host-only seguras.

### P1 — El chofer no encuentra cargas públicas en su listado

El listado del chofer presentó “No hay cargas” aunque había cargas públicas en estado publicada; el publicador sí las veía y el chofer pudo acceder mediante URL directa. Esto impide el flujo normal de encontrar y aceptar trabajo. Revisar consulta por `driver_board_id`, audiencia, empresa y filtros por defecto contra los datos del despliegue.

### P1 — La valoración se anuncia pero no se puede enviar

El detalle entregado muestra el formulario de valoración, pero el POST falla con 405 en la instancia activa. Sin valoración no puede comprobarse el promedio ni el ordenamiento solicitado por producto.

### P1 — Perfil roto

Ambos endpoints de perfil responden 500. Además de impedir editar datos, bloquea la subida de foto y la revisión del promedio del chofer.

### P2 — Recuperación de contraseña es un enlace muerto

Login ofrece “¿Olvidaste tu contraseña?”, pero no existe flujo operativo y el enlace regresa al login. Retirar el enlace hasta implementar recuperación segura o completar el flujo con tokens de un solo uso y expiración.

### P2 — Validación de correo no comprueba que el dominio/buzón sea real

Los registros QA con dominio reservado `.invalid` fueron aceptados. En el código, `EmailService.EmailExists` tiene desactivado el verificador dentro de `if false` y retorna `true`; el paso de registro que afirma comprobar el dominio no está haciendo esa verificación. La sintaxis básica sí se valida. Implementar verificación por correo con token, evitando depender de sondas SMTP como requisito de registro.

### P2 — La audiencia privada quedó inconclusa

El envío QA con audiencia `red_privada` creó temporalmente una carga en la instancia. Se eliminó enseguida por la interfaz. Como `/red-choferes` no funciona y el detalle no expone la audiencia, no se pudo demostrar si el registro pertenecía a una red con miembros ni si choferes ajenos podían verla. Repetir únicamente en staging cuando el backend y la red estén operativos; comprobar visibilidad con una cuenta miembro y otra ajena.

## Datos QA que permanecen

En el listener local `127.0.0.1:3000` se dejaron tres usuarios QA activos, todos con nombres que comienzan por `BUFALO QA` y correos `bufalo.qa.*.20261004@example.invalid`: dos choferes y un publicador. También quedan las empresas QA (IDs observados 4 y 5), las direcciones de prueba (IDs 7 y 8), las cargas entregadas (IDs 7 y 8) y una factura pagada (ID 3). No se confirmó que esos datos existan en la BD del dominio público. Se usaron exclusivamente formularios web; la dirección temporal ID 9 y la carga privada temporal ID 9 se eliminaron por sus formularios. No se guardaron contraseñas en este informe.

## Próximos pasos para cerrar la prueba

1. Alinear backend, migraciones, assets y plantillas desde el mismo commit; exponer el SHA/build activo y comprobar `/profile`, `/choferes`, `/red-choferes` y POST de rating.
2. Corregir el listado de cargas para que el chofer vea cargas `load_board` disponibles, y probar filtros/paginación con al menos una carga pública y una privada.
3. Repetir perfil, foto, red, rating y búsqueda ordenada con usuarios QA en staging; probar entradas inválidas, duplicados y acceso cruzado.
4. Probar administrativamente alta/edición/desactivación de usuarios, empresas, cargas y facturas con una cuenta admin QA aprobada.
5. Probar edición/cancelación de factura, vencimiento, idempotencia de emisión/pago y duplicado por carga; comprobar concurrencia e integridad financiera.
6. Ejecutar la suite completa con PostgreSQL/Redis de pruebas y auditoría de dependencias con acceso al registro. Validar luego los mapas y flujos móviles en navegador/emulador real.
7. Limpiar los usuarios, empresas, direcciones, cargas y factura QA del despliegue cuando se autorice cerrar esta campaña.

**Dictamen:** el servicio del puerto 3000 no está listo para producción funcional. Los flujos base de registro, dirección, transición de carga y ciclo principal de factura pasan, pero hay bloqueos P1 y una discrepancia de despliegue que invalida cualquier declaración de cobertura completa.
