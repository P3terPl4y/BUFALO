# BUFALO: mapa para desarrollo y mantenimiento

Revisión del 8 de octubre de 2026 sobre el árbol local, cuyo HEAD es `ddeea6f`.
Existen numerosos cambios modificados y sin seguimiento anteriores a esta revisión.
Este documento describe ese árbol, no una release ni el estado de producción.
No se leyó `.env`, no se ejecutaron migraciones ni se modificaron datos operativos.

## Arquitectura y puntos de entrada

Monolito Go, módulo `goravel`, con Goravel 1.18 y servidor HTTP Fiber v3.5.
`go.mod` declara Go 1.25.0; el entorno de esta revisión usa Go 1.26.0.
No hay una SPA activa: HTML renderizado con `github.com/gofiber/template/html/v3`.
PostgreSQL se accede mediante el ORM de Goravel/GORM. Redis almacena sesiones y
contadores distribuidos de autenticación. SMTP entrega confirmaciones de registro.

El recorrido principal es:

`main.go → bootstrap/config → middleware Fiber → routes/web.go → controlador → servicio/ORM → plantilla`.

- `main.go`: bootstrap, comandos artisan, validación productiva, administrador
  opcional, Redis, plantillas, seguridad, sesiones, CSRF, probes y archivos estáticos.
- `production_config.go`: valida configuración de producción antes del alta admin.
- `bootstrap/migrations.go`: registro explícito de las 13 migraciones actuales.
- `routes/web.go`: rutas web y autorización por rol. No existe `routes/api.go`.
- `app/facades`: adaptadores del framework; no confundirlos con reglas del dominio.
- `app/http/controllers`: entrada HTTP, permisos por objeto, formularios y vistas.
- `app/services`: persistencia y operaciones reutilizables. Hay consultas directas
  en controladores, especialmente dashboards y administración; la separación no es completa.
- `app/billing`, `app/community`: reglas independientes de HTTP y persistencia.
- `app/exports`: CSV, XLSX y PDF, con protección contra fórmulas en CSV.
- `app/views/layouts/base.html`, `public/css/style.css`: interfaz compartida.
- `public/js/load-route.js`: mapas/rutas; `admin.js` y `admin-metrics.js`: panel.

`mobile/` contiene restos locales ignorados: la aplicación móvil fue retirada en
HEAD y no aparece en `git ls-files mobile`. No tratar esos restos como producto activo.
No se encontró un workflow CI versionado en `.github/`.

## Dominio y reglas que deben preservarse

Roles: `admin`, `publicador`, `chofer`. Las empresas admiten tipos broker, carrier,
shipper, factoring y mixto. Usuario y perfil profesional tienen IDs distintos;
no usar `users.id` como si fuera `chofers.id` o `publicadors.id`.

Los nombres SQL reales incluyen `chofers`, `publicadors` y `direccions`, además de
`users`, `empresas`, `cargas`, `facturas`, `cargas_historial`, `red_choferes`,
`chofer_calificaciones` y `pending_registrations`. Confirmarlos en migraciones,
no deducirlos de los títulos en español.

### Registro y autenticación

- `AuthController`, `UserService` y `EmailVerificationService` implementan el alta.
- El registro guarda un payload cifrado temporal y el hash SHA-256 de un token
  aleatorio; el enlace expira en 24 horas. GET presenta confirmación; POST crea
  usuario, empresa opcional y perfil en una transacción y consume el token.
- `SessionAuth` consulta el usuario en DB en cada petición protegida y actualiza
  rol/actividad; desactivar la cuenta revoca el acceso de sesiones existentes.
- `RoleAuth` limita roles; los controladores deben verificar también ownership.
- Sesiones Redis: 30 minutos de inactividad y 24 horas absolutas. En producción,
  cookie `__Host-session`, Secure y HttpOnly; CSRF asociado a sesión.
- Login: 10 solicitudes/IP/minuto. Registro: 5/IP/15 minutos y 2/email/hora.
  Contadores Redis atómicos con Lua y claves HMAC. Redis caído produce 503 en
  estas rutas; no se sustituye silenciosamente por memoria.

### Cargas, red y calificaciones

- Ruta principal `/loads`; controladores `CargaController` y `LoadIntersetController`
  (el nombre del archivo contiene ese error ortográfico).
- Flujo operativo: `publicada → asignada → en_transito → entregada`.
  Los enums también incluyen `negociando` y `cancelada`; ello no significa que
  todas sus transiciones tengan un caso de uso implementado.
- Aceptación/asignación usan actualizaciones condicionales y filas afectadas para
  impedir doble asignación. Tránsito y entrega exigen chofer asignado y estado previo.
- Audiencias operativas: `load_board` y `red_privada`. `red_extendida` existe
  como enum, pero `community.ValidateAudience` la rechaza.
- La red relaciona empresa y chofer; las valoraciones requieren carga entregada,
  publicador propietario y una sola valoración por carga. El agregado del chofer
  se recalcula dentro de la transacción de valoración.
- Fotografías: JPEG/PNG, hasta 5 MB y límites de dimensiones/píxeles.

### Facturas

- `FacturaController` aplica permisos y filtros; `FacturaService` persiste;
  `billing` valida importes, ownership y transiciones.
- Una factura por carga, protegida por índice único. La carga debe estar entregada.
- Acceso por perfiles vinculados; escritura para el perfil emisor o admin.
  Pertenecer a la misma empresa no concede automáticamente acceso.
- Creación en borrador; solo borradores editables. Emitir, pagar, vencer y cancelar
  tienen transiciones explícitas. Eliminar conserva la fila mediante cancelación.
- Actualizaciones de estado incluyen estado previo para detectar concurrencia.
- Monedas CUP/MLC/USD/EUR. Total derivado de subtotal más impuestos; redondeo
  decimal a centavos en `billing`, aunque modelos y formularios aún usan float64.
- No asumir pagos parciales, conciliación, numeración fiscal ni vencimiento
  automático. La exportación PDF sí existe actualmente.

## Operación

`go run .` inicia HTTP; `go run . artisan migrate` ejecuta migraciones explícitas.
No se ejecutaron estos comandos durante el estudio porque pueden escribir en DB.
El administrador bootstrap solo se crea si se configuran email y contraseña;
no hay credenciales admin predeterminadas en el arranque actual.

- `/` sirve la landing de `public/`; no redirige siempre a login.
- `/healthz`: liveness. `/readyz`: SELECT 1 y PING Redis, plazo total de 2 segundos.
  Ambos omiten sesión/CSRF; readiness devuelve 503 genérico si falla una dependencia.
- `/admin/health`: métricas locales al proceso y comprobaciones TCP. TCP alcanzable
  no equivale a consulta DB autenticada. El p95 es aproximado por intervalos.
- `config/database.go` admite `DB_DSN` y pool acotado: 25 abiertas, 10 inactivas,
  300 segundos de inactividad y 1800 segundos de vida por defecto.
- No hay replay automático de escrituras ante fallos DB; multi-host depende de HA
  real del proveedor y no crea replicación por sí mismo.
- Docker ejecuta usuario sin privilegios, raíz de solo lectura y volumen de avatares.
  Compose no crea PostgreSQL ni Redis. Alternativa: `deploy/bufalo.service`.
- El proxy confiable actual es loopback. Comprobar esa topología al desplegar.

Referencia operativa principal: `docs/DEPLOY_PRODUCCION.md`. Para pendientes de
staging y plataforma: `docs/PRODUCTION_READINESS_PLAN_2026-10-06.md`.

## Pruebas y límites

Las suites de servicios/controladores/feature/redteam contienen escrituras,
migraciones y TRUNCATE. Exigen `APP_ENV=testing`, `DB_DATABASE` terminado en
`_test` y coincidencia con DB/DSN efectivos. Usar una base independiente por
paquete que ejecuta migraciones; `-p 1` por sí solo no evita interferencia de esquema.
Las pruebas feature usan sesión en memoria y no reproducen todo el stack CSRF/Redis.

Comando de revisión sin base PostgreSQL de integración:

```bash
go test . ./app/billing ./app/community ./app/dbresilience ./app/exports ./app/models ./app/monitoring ./app/http/middleware ./app/viewhelpers ./routes -count=1
```

Resultado del 8 de octubre: comando aprobado en los nueve paquetes con tests;
`routes` compila y no contiene tests. Las pruebas Redis pueden omitirse si el sandbox impide sockets;
el resultado general no demuestra que se haya ejecutado Redis real.
No se ejecutó la suite completa ni se verificó producción durante este estudio.

## Hallazgos para el siguiente trabajo

1. **Privacidad del dashboard:** `AuthController.ShowHome` consulta las primeras
   20 cargas publicadas sin filtrar audiencia, propietario ni red. `home.html`
   muestra referencia, ciudades, fecha, peso, equipo, distancia y tarifa. El detalle
   y listado sí tienen reglas de visibilidad. Hallazgo por lectura del código;
   falta prueba HTTP que demuestre y cubra el caso privado desde `/home`.
2. **Documentación desfasada:** `DEVELOPMENT.md` menciona Go 1.22, `.env.example`,
   `demo_data.go`, admin predeterminado y raíz que redirige a login. No describe
   fielmente el árbol actual. La auditoría de facturación y el plan de abuso
   contienen pendientes ya implementados (transacciones, revocación, correo, PDF).
   Leer fechas y actualizaciones; no reimplementar por una lista histórica.
3. **Integridad y concurrencia:** revisar restricciones SQL FK/CHECK, auditoría de
   operaciones, versiones de borradores y consistencia de agregados de rating
   bajo escrituras concurrentes. Tags GORM no demuestran restricciones instaladas.
4. **Consistencia del login:** el alta normaliza email, pero `HandleLogin` consulta
   el texto recibido sin normalización explícita. Verificar login con espacios
   y mayúsculas en una prueba antes de definir el cambio.
5. **Entrega:** el árbol está mezclado con cambios previos; revisar el diff y
   preparar una release concreta antes de desplegar. Faltan verificaciones propias
   actuales de navegador, SMTP, fallos de dependencias, restauración y capacidad.

## Cómo continuar

Para cada cambio seguir ruta, middleware, permiso por objeto, controlador,
servicio/regla, migración si procede y vista. Mantener textos visibles en español,
plantillas en `layouts/base` y estilos compartidos. Registrar migraciones nuevas
en bootstrap; evitar modificar migraciones históricas ya aplicadas. Elegir pruebas
de comportamiento para permisos, estados y concurrencia. Conservar los cambios
locales ajenos y actualizar este mapa cuando cambien los contratos aquí descritos.
