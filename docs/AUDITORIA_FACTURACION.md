# Revisión crítica de BUFALO y facturación

Fecha: 2 de octubre de 2026. Alcance: código local, arquitectura, rutas, servicios, modelos, migraciones, vistas y suites disponibles. Esta revisión no certifica cumplimiento fiscal ni demuestra que cada comportamiento posible del sistema esté libre de errores.

## Arquitectura observada

Monolito Go con Goravel para configuración, ORM y migraciones; Fiber para HTTP, sesiones, CSRF y middleware; PostgreSQL y vistas HTML del servidor. Redis conserva las sesiones en la aplicación real. Los controladores concentran validación, consultas y autorización. Las reglas de negocio estaban mezcladas con persistencia y existían discrepancias entre roles de usuario (`publicador`, `chofer`) y tipos de empresa (`broker`, `carrier`).

Se incorpora `app/billing` como paquete de reglas independiente del servidor y la base. Es un primer paso, no una reestructuración completa: los servicios siguen dependiendo de facades globales y la autorización administrativa general continúa en middleware.

## Problemas comprobados y cambios

| Prioridad | Problema previo | Cambio y límite |
|---|---|---|
| Crítica | Facturas comparaban empresa con ID de usuario; listado usaba roles de empresa y no de usuario. Edición, pago y cancelación no comprobaban pertenencia. | Lectura limitada a empresa emisora para publicador y receptora para chofer; escritura solo empresa emisora o admin. Usuario, rol, empresa y actividad se consultan en BD. La autorización es por empresa, no por creador individual. |
| Crítica | Total recibido del navegador; impuestos negativos, NaN, infinito y fechas incoherentes no se rechazaban sistemáticamente. | Validación común en servicio; total derivado del subtotal más impuestos; centavos y redondeo decimal half-up. Se conserva float64 en modelos por compatibilidad: conviene migrar la representación del dominio a decimal o unidades menores. |
| Alta | Pago directo de borradores y sobreescritura de pagos existentes. | Máquina de estados, fecha y método de pago completos; rechazo de transiciones inválidas y protección contra concurrencia mediante estado en WHERE y filas afectadas. No demuestra ausencia de todas las carreras posibles. |
| Alta | Borrado físico de documentos de facturación. | Cancelación conserva fila e identidad; facturas pagadas no se editan ni cancelan. No hay notas de crédito ni devoluciones. |
| Alta | Creación sin comprobar relación carga/empresas/conductor. | Requiere carga entregada, emisor igual a empresa de carga y receptor igual a empresa del chofer asignado. Chofer se deriva de carga. La política de facturar solo cargas entregadas es una decisión conservadora de esta implementación. |
| Alta | No existían `facturas/index`, `show`, `create`, `edit`. | Se crean vistas y se incorpora emisión explícita. Interfaz básica; faltan selección guiada y exportación. |
| Crítica | La sesión confiaba en estado y rol almacenados, incluso tras revocación del usuario; además no se verificaba el error de renovar sesión al autenticar. | El middleware consulta la cuenta en BD, refresca el rol, revoca sesiones de cuentas desactivadas/eliminadas y responde 503 sin borrar sesión si falla la BD; login falla cerrado ante error de regeneración. Pruebas funcionales cubren degradación y desactivación. |
| Alta | Direcciones dereferenciaban `OwnerID` sin validar nulos y respondían éxito cuando un usuario ajeno no podía borrarlas. | Comprobación común owner/admin, denegación segura para datos legacy sin owner y validación/rango de coordenadas para alta/edición. Pruebas unitarias de autorización. |
| Crítica | Administrador creado con email y contraseña constantes conocidos. | Arranque solo crea admin si se configuran `BOOTSTRAP_ADMIN_EMAIL` y `BOOTSTRAP_ADMIN_PASSWORD`; mínimo 12 caracteres. No modifica credenciales de administradores ya existentes. |
| Crítica | Tests destructivos sin aislamiento; `migrate:fresh` devolvía éxito sin crear tablas. | Exigen APP_ENV=testing y DB_DATABASE explícito terminado en `_test`, coincidente con configuración activa. Crean esquema mediante migraciones registradas; limpieza exige esa comprobación. Ejecutar paquetes secuencialmente con `-p 1`: comparten base. |
| Alta | Limpieza truncaba tablas inexistentes; errores se registraban y las pruebas seguían. | Nombres corregidos a `chofers`, `publicadors`, `direccions`, `cargas`; errores fallan inmediatamente. |
| Media | Funciones de plantillas disponibles en producción no existían en pruebas. | Registro compartido `app/viewhelpers` para ambos entornos y prueba de carga de todas las vistas. |
| Media | Rate limiter contaminaba casos funcionales; pruebas de registro/admin desactualizadas. | Override explícito del limiter en harness funcional y prueba separada del sexto intento; formularios actualizados a roles reales. Producción conserva limiter por defecto. |
| Media | Filtro de fecha incluía medianoche del día posterior; paginación admitía valores negativos o ilimitados. | Límite superior exclusivo y normalización de tamaño hasta 100 registros en servicio. Controlador público normaliza también sus parámetros; el controlador admin todavía muestra los originales. |
| Media | Verificación estática fallaba al formatear punteros float como texto con `%q`. | Formato `%v` en diagnóstico de direcciones. |

## Ciclo implementado

1. Publicador/admin crea borrador para carga entregada con empresas coincidentes. Número de factura y carga mantienen unicidad por los índices existentes.
2. Solo borradores admiten actualización de importes y moneda. Total se recalcula en servidor; empresa, carga y número permanecen estables.
3. `POST /facturas/:id/emitir` pasa borrador a emitida.
4. Emitida o vencida admite registro de un pago completo, método y fecha. Repetir el pago falla en servicio.
5. Borrador, emitida o vencida admite cancelación; se conserva el registro. Pagada y cancelada son terminales.
6. El dominio permite emitida → vencida cuando vence. Todavía no hay tarea programada ni ruta pública para automatizar esa transición.

El usuario debe tener empresa asociada para facturar. El formulario solicita IDs explícitos; errores de servicio se muestran de forma genérica. No existen pagos parciales, conciliación bancaria, factura PDF, envío de correo, numeración fiscal automática, conceptos múltiples, porcentajes fiscales configurables ni historial de eventos.

## Hallazgos pendientes por prioridad

1. **Integridad de BD:** migraciones crean índices pero no declaran claves foráneas ni CHECK para importes, monedas y estados. Tags de GORM no sustituyen restricciones en migraciones. Es necesaria una migración nueva, tras revisar datos existentes; no se aplicaron cambios a BD operativa.
2. **Transacciones:** `UserService.CreateWithRole` hace rollback manual ignorando errores. Debe usar transacción real para usuario, empresa, perfil y vínculos. La asignación manual de chofer no limita estado ni comprueba filas afectadas.
3. **Seguridad transversal:** revisar permisos por objeto en cargas, empresas, perfiles y direcciones. El middleware de actividad general usa valor almacenado en sesión; puede quedar desactualizado tras bloquear al usuario. Facturación reconsulta actividad, otros módulos necesitan el mismo criterio.
4. **Auditoría:** añadir actor, hora, motivo y versiones de cambios a facturas y eventos de pago/cancelación. La conservación de filas implementada no equivale a auditoría inmutable.
5. **Consistencia concurrente:** se protege cambio de estado de factura, pero no se bloquea carga/empresa/chofer durante creación. Cambios simultáneos pueden producir relaciones obsoletas. Ediciones paralelas de un mismo borrador pueden sobrescribirse: falta versión optimista.
6. **Modelo monetario:** migrar float64 a decimal o centavos; definir precisión efectiva de distancia y tarifa en BD. No se calcula automáticamente subtotal desde distancia por tarifa: subtotal es un importe negociado editable.
7. **Experiencia y errores:** selección de cargas elegibles y empresas, validación detallada, filtros estrictos y conservación de filtros al cambiar de página, confirmación visible de emisión/cancelación. Admin todavía conserva algunos textos de borrado y enlaces de edición heredados.
8. **Facturación completa:** vencimiento automático, notas de crédito, devoluciones, pagos parciales, comprobantes, reintentos idempotentes, conciliación y numeración según requisitos comerciales. Falta definición del usuario sobre estos comportamientos. Se conserva la convención del modelo existente (empresa del publicador como emisor y empresa del chofer como receptor); antes de usar documentos comerciales reales debe confirmarse quién presta el servicio, quién cobra y quién paga.
9. **Pruebas:** añadir navegación de navegador, CSRF real, sesiones Redis, revocación de roles, concurrencia SQL, base por paquete y fallos de infraestructura. Las pruebas HTTP nuevas usan sesión en memoria y no incluyen CSRF de producción.
10. **Operación:** revisar secretos fuera de repositorio, rotación de credenciales, backups/restauración, logs sin datos innecesarios, métricas y manejo consistente de errores.

## Verificación reproducible

Base PostgreSQL exclusiva, UTF-8, cuyo nombre termina en `_test`. Nunca usar una base compartida u operativa. Las suites truncan tablas y deben ejecutarse de forma secuencial.

```sh
APP_ENV=testing DB_CONNECTION=postgres DB_HOST=127.0.0.1 DB_PORT=55441 DB_DATABASE=bufalo_validation_test DB_USERNAME=peter DB_PASSWORD='' go test -p 1 ./... -count=1 -timeout=120s
go test ./app/billing -race -cover
go test ./... -run '^$' -exec /bin/true
```

El último comando compila y ejecuta la comprobación estática sin ejecutar los binarios de tests; no demuestra comportamiento. Los resultados finales se registran debajo. Los cambios previos del usuario en `go.mod`, `go.sum` y archivos de logs se conservaron.

## Alcance real de cobertura

| Área | Evidencia disponible |
|---|---|
| Facturación | Reglas unitarias: importes y half-up, fechas, monedas, acceso y matriz de 25 transiciones. Integración PostgreSQL: creación, unicidad, edición, emisión, pago, inexistentes y cancelación preservando registro. HTTP: acceso ajeno rechazado y consulta/edición/emisión/pago del emisor. |
| Usuarios y alta profesional | Suites existentes de servicio y registro HTTP; pruebas de duplicados, perfiles, empresa y rollback manual. No demuestran atomicidad bajo fallos del servidor o concurrencia. |
| Login, perfiles, administración de usuarios | Suites funcionales existentes; limitador probado por separado. |
| Vistas | Todas las plantillas se cargan con el mismo registro de funciones que producción. Esto no prueba diseño visual ni cada rama de renderizado. |
| Cargas, empresas, direcciones y perfiles profesionales | Código revisado y compilado. No existe una suite exhaustiva de cada endpoint y transición; no se declara cobertura funcional completa de estos módulos. |
| Redis, navegador y CSRF real | Pendientes; los harness HTTP usan memoria y no reproducen toda la infraestructura de producción. |

## Resultados finales

- Suite completa: `go test -p 1 ./... -count=1 -timeout=120s`, código de salida 0 en PostgreSQL 18 aislado (`bufalo_validation_test`, UTF-8, puerto 55441). Todos los paquetes con pruebas pasaron, incluidas las suites HTTP y de persistencia. Se añadieron casos de sesión degradada/desactivada y propiedad de direcciones.
- `go test ./app/billing -race -cover`: aprobado, 94,2 % de sentencias del paquete de reglas de facturación. Esta cifra no es cobertura del proyecto completo.
- `go test ./app/viewhelpers ./app/billing ./app/http/middleware -race -cover`: aprobado. Sin carreras detectadas en esos paquetes durante estas pruebas; persistencia y toda la aplicación no se ejecutaron con `-race`.
- Compilación y comprobación estática inicial de todos los paquetes: aprobadas. La suite completa final incluye comprobación estática por defecto.
- `go vet ./...`: aprobado sin diagnósticos.
- `git diff --check`: aprobado.
- Log de la ejecución completa final: `/tmp/bufalo-final-pass.log`. La base de pruebas aislada permanece en `/tmp/bufalo-audit-pg`; el servidor PostgreSQL se detiene tras validar.

Se corrigió durante la revisión un error introducido en la plantilla de paginación que inicialmente impedía cargar todas las vistas. La nueva prueba `TestAllTemplatesLoad` verifica este tipo de fallo. No se presenta una ejecución anterior fallida como éxito ni se ocultaron pruebas mediante skips.
