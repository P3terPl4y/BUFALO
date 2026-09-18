# Guía de desarrollo de BUFALO

Esta guía explica cómo está organizado el proyecto, cómo fluye una petición y dónde debe trabajar un desarrollador cuando agrega una funcionalidad.

## 1. Qué es BUFALO

BUFALO es una aplicación web de gestión de cargas. Un **publicador** registra cargas, rutas, tarifas y empresas; un **chofer** consulta cargas y puede aceptar o mostrar interés; el **administrador** supervisa usuarios y operaciones. La interfaz se renderiza en el servidor con plantillas HTML y el estado de autenticación se conserva en sesión.

El proyecto es un monolito Go. La aplicación, las vistas, las migraciones y los servicios viven en el mismo repositorio. PostgreSQL almacena los datos y Redis almacena las sesiones.

## 2. Arranque de una petición

1. `main.go` inicializa Goravel, carga `.env`, registra el ORM, configura Redis, sesiones, CSRF, Helmet, logs, archivos estáticos y el motor de plantillas.
2. `routes/web.go` registra las rutas públicas y protegidas.
3. Las rutas protegidas pasan por `SessionAuth`, que lee `user_id` desde la sesión y lo coloca en `ctx.Locals`.
4. Los middlewares de rol (`AdminAuth`, `PublicadorAuth`, `ChoferAuth` y `RoleAuth`) autorizan la operación.
5. El controlador valida la entrada, llama a un servicio y decide la vista o redirección.
6. El servicio consulta PostgreSQL mediante `facades.Orm()` y devuelve modelos o errores.
7. La plantilla se renderiza dentro de `app/views/layouts/base.html`, que contiene el header, sidebar, avisos globales, footer y scripts comunes.

El flujo recomendado es **ruta → middleware → controlador → request/modelo → servicio → vista**. Las consultas de negocio deben permanecer en `app/services`, no en las plantillas.

## 3. Mapa del repositorio

| Ruta | Responsabilidad |
|---|---|
| `main.go` | Arranque, configuración de infraestructura y servidor HTTP |
| `routes/web.go` | Rutas web, grupos de autenticación y permisos |
| `bootstrap/` | Proveedores y lista ordenada de migraciones |
| `config/` | Configuración de base de datos, sesión, correo, cache y aplicación |
| `app/http/controllers/` | Casos de uso HTTP y redirecciones |
| `app/http/middleware/` | Sesión, autorización, CSRF y límites de login |
| `app/models/` | Entidades persistidas y relaciones |
| `app/requests/` | Estructuras para recibir formularios y JSON |
| `app/services/` | Consultas, reglas de negocio y operaciones reutilizables |
| `app/views/` | Plantillas HTML organizadas por módulo |
| `app/views/layouts/base.html` | Shell compartido de las páginas autenticadas |
| `public/css/style.css` | Design system y estilos de la aplicación |
| `public/js/` | Comportamiento del frontend que no pertenece a una vista concreta |
| `database/migrations/` | Creación y evolución del esquema PostgreSQL |
| `demo_data.go` | Datos de demostración para desarrollo local |
| `tests/` | Pruebas unitarias y funcionales |

## 4. Módulos y rutas

- **Autenticación:** `/login`, `/register`, `/logout`.
- **Inicio:** `/home`. El dashboard reúne métricas y accesos operativos.
- **Cargas:** `/loads`, `/loads/create`, `/loads/:id`, edición, eliminación, aceptación e interés.
- **Direcciones:** `/direcciones`, creación, edición y detalle. La grilla usa 10 registros por página.
- **Empresas:** `/empresas` y sus operaciones CRUD.
- **Choferes y publicadores:** perfiles operativos y edición según el rol.
- **Facturas:** consulta, creación, edición y marcado como pagada.
- **Perfil:** `/profile` y `/profile/edit`.
- **Administración:** `/admin/users`, restringido a administradores.

Para agregar una ruta, primero registra el verbo y URL en `routes/web.go`, luego aplica el middleware de rol mínimo, implementa el método del controlador y crea la plantilla. Si la operación se reutiliza o consulta datos, extrae esa parte a un servicio.

## 5. Roles y autorización

Los roles actuales son `admin`, `publicador` y `chofer`.

- `SessionAuth` exige una sesión válida.
- `RoleAuth("publicador", "chofer", "admin")` permite una lista explícita de roles.
- `AdminAuth` limita el grupo `/admin`.
- Las reglas de ownership deben comprobar el usuario actual antes de editar o eliminar un recurso.

No debe confiarse en el rol enviado por un formulario. El rol se obtiene de la sesión y de la base de datos. Si una nueva acción modifica datos, debe tener middleware de escritura y validación de ownership.

## 6. Base de datos

Las migraciones se declaran en `bootstrap/migrations.go` y se ejecutan en ese orden. Las tablas principales son `users`, `chofer`, `publicador`, `empresas`, `cargas`, `facturas`, `cargas_historial` y `direcciones`.

Para una instalación existente:

```powershell
Set-Location D:\xampp2\BUFALO
go mod download
go run . artisan migrate
```

`migrate:fresh` elimina y vuelve a crear las tablas; úsalo solo en desarrollo. Después puedes cargar los datos de demostración con el flujo documentado en `README.md` y `demo_data.go`.

La conexión se define en `.env`. Nunca se debe subir `.env` ni incluir contraseñas reales en commits. `.env.example` sirve como plantilla.

## 7. Frontend y plantillas

Las páginas autenticadas deben usar `layouts/base`. El layout centraliza navegación, menú de usuario, tema claro/oscuro, loader, avisos y footer. Las vistas de módulo solo deben renderizar su contenido.

Los estilos se mantienen en `public/css/style.css`. Las reglas nuevas deben colocarse junto al componente que modifican o en la sección de overrides final, usando nombres `bf-` para el shell BUFALO y `tf-` para componentes de la interfaz operativa. Evita estilos inline y dependencias nuevas si el componente se puede resolver con el design system existente.

Los mensajes de resultado se envían como `flash_success` o `flash_error` en la redirección. El layout los muestra y los cierra automáticamente; una vista nueva no debe crear un segundo toast para el mismo mensaje.

## 8. Configuración local

Requisitos: Go 1.22+, PostgreSQL y Redis. La base de datos puede ser una instancia local o un proyecto PostgreSQL administrado como Supabase.

```powershell
Set-Location D:\xampp2\BUFALO
Copy-Item .env.example .env
# Edita .env con PostgreSQL y Redis
go mod download
go run . artisan migrate
go run .
```

La aplicación queda en `http://localhost:3000`; `/` redirige a `/login`. El usuario administrativo de desarrollo creado por el arranque es `admin@example.com` con contraseña `Admin123!`. Cambia esa contraseña antes de usar datos reales.

Para detener el servidor en la terminal que lo ejecuta, usa `Ctrl+C`. Si está ejecutándose en otra terminal de Windows, localiza el proceso y ciérralo desde el Administrador de tareas o con `Stop-Process` tras verificar su PID.

## 9. Pruebas y verificación

```powershell
go test .
go test ./...
git diff --check
```

Una revisión funcional mínima debe comprobar login, redirección de `/`, acceso por rol, creación y edición de una carga, paginado de grillas, avisos flash y cierre de sesión. Si `go test ./...` falla por falta de espacio en el disco durante el linking, libera espacio temporal y repite; no confundas ese error de entorno con un fallo de compilación del código.

## 10. Convenciones para nuevos cambios

- Usa nombres de métodos y variables descriptivos en inglés, y textos visibles en español.
- Valida formularios en el controlador antes de llamar al servicio.
- Devuelve errores controlados y mensajes flash útiles; no expongas SQL ni secretos al usuario.
- Mantén la paginación en el servicio y pasa a la vista `page`, `perPage`, `totalPages`, `hasPrev` y `hasNext`.
- Agrega comentarios solo cuando expliquen una decisión no obvia.
- Prueba primero en local y revisa el diff antes de hacer commit.
- No edites `.env`, `storage/` ni datos generados para incluirlos en Git.

## 11. Diagnóstico rápido

**`relation "users" does not exist`:** ejecuta `go run . artisan migrate` contra la misma base configurada en `.env` y confirma el esquema `public`.

**El login vuelve a `/login`:** revisa Redis, la cookie de sesión y que el servidor y el navegador usen el mismo puerto.

**Una plantilla muestra texto extraño como `` `n ``:** busca el marcador literal en `app/views` con `rg -n -F '`n' app/views`.

**Una página responde 403:** revisa el middleware de rol y el rol guardado en `users`.

**La grilla no pagina:** confirma que el controlador usa `GetPage`, que `perPage` vale 10 y que la URL conserva el parámetro `page`.

