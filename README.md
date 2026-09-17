# 🐃 BUFALO
Plataforma de gestión logística y marketplace de cargas para Cuba, inspirada en DAT One. Conecta **publicadores** (brokers) con **choferes** (carriers) para publicar, negociar y asignar cargas.

---

## 📑 Tabla de contenidos

1. [Descripción general](#-descripción-general)
2. [Stack tecnológico](#-stack-tecnológico)
3. [Estructura del proyecto](#-estructura-del-proyecto)
4. [Modelos de datos](#-modelos-de-datos)
5. [Roles y permisos](#-roles-y-permisos)
6. [Rutas principales](#-rutas-principales)
7. [Frontend — Design System BUFAL](#-frontend--design-system-bufal)
8. [Arquitectura del CSS](#-arquitectura-del-css)
9. [Instalación y ejecución](#-instalación-y-ejecución)
10. [Testing](#-testing)

---

## 🎯 Descripción general

BUFALO es un **clon funcional de DAT One adaptado al mercado cubano**:

- **Publicadores** (brokers) publican cargas con origen, destino, peso, tarifa y tipo de equipo.
- **Choferes** (carriers) buscan cargas, se interesan, o las aceptan directamente.
- **Empresas** (broker o carrier) agrupan usuarios y permiten operar en red.
- **Direcciones** se seleccionan en un mapa interactivo (Leaflet + Nominatim).
- **Ownership estricto**: cada usuario solo puede editar sus propios recursos.

Características clave:

- ✅ Registro con 3 modos de empresa: nueva, existente, sin empresa.
- ✅ Mapa interactivo para crear direcciones (búsqueda + clic + drag).
- ✅ Roles diferenciados (admin / publicador / chofer).
- ✅ Ciclo de vida completo de la carga.
- ✅ Design system propio con tema claro/oscuro.
- ✅ Responsive y accesible.

---

## 🛠 Stack tecnológico

| Capa | Tecnología |
|---|---|
| **Lenguaje** | Go 1.22+ |
| **Framework** | [Goravel](https://goravel.dev) + [Fiber v3](https://docs.gofiber.io) |
| **ORM** | GORM (vía Goravel) |
| **Base de datos** | PostgreSQL |
| **Sesiones** | Redis (`github.com/gofiber/storage/redis/v3`) |
| **Templates** | `gofiber/template/html/v3` |
| **Mapas** | Leaflet 1.9.4 + `leaflet.geodesic` |
| **Geocoding** | Nominatim (OpenStreetMap) |
| **Frontend CSS** | Design System propio (BUFAL v7) |
| **Iconos** | Bootstrap Icons 1.11.3 |
| **Testing** | `go test` + scripts bash con `curl` |

---

## 📂 Estructura del proyecto

```
DATClone/
├── main.go                          # Bootstrap de la app, middlewares, arranque
├── go.mod
├── go.sum
│
├── app/
│   ├── facades/                     # Facades de Goravel (Orm, Hash, Validation…)
│   ├── http/
│   │   ├── controllers/             # Todos los controllers HTTP
│   │   │   ├── AuthController.go
│   │   │   ├── CargaController.go
│   │   │   ├── ChoferController.go
│   │   │   ├── DireccionController.go
│   │   │   ├── EmpresaController.go
│   │   │   ├── FacturaController.go
│   │   │   ├── PublicadorController.go
│   │   │   ├── UserController.go
│   │   │   ├── AdminController.go
│   │   │   ├── LoadInterestController.go
│   │   │   └── helpers.go           # strPtr, etc.
│   │   └── middleware/
│   │       ├── session_auth.go      # SessionAuth
│   │       ├── role_auth.go         # RoleAuth, PublicadorAuth, ChoferAuth, AdminAuth
│   │       └── login_rate_limiter.go
│   │
│   ├── models/                      # Modelos GORM
│   │   ├── user.go
│   │   ├── empresa.go
│   │   ├── chofer.go
│   │   ├── publicador.go
│   │   ├── carga.go
│   │   ├── factura.go
│   │   ├── direccion.go
│   │   └── enums.go                 # Tipos: TipoEmpresa, EstadoCarga, etc.
│   │
│   ├── requests/                    # Structs para bindear formularios
│   │   ├── user_requests.go
│   │   ├── carga_requests.go
│   │   ├── direccion_requests.go
│   │   ├── empresa_requests.go
│   │   ├── chofer_requests.go
│   │   ├── publicador_requests.go
│   │   └── factura_requests.go
│   │
│   ├── services/                    # Lógica de negocio
│   │   ├── UserService.go
│   │   ├── CargaService.go
│   │   ├── DireccionService.go
│   │   ├── EmpresaService.go
│   │   ├── ChoferService.go
│   │   ├── PublicadorService.go
│   │   ├── FacturaService.go
│   │   └── EmailService.go
│   │
│   └── views/                       # Templates HTML
│       ├── layouts/
│       │   └── base.html            # Layout base con navbar + embed
│       ├── auth/
│       │   ├── login.html
│       │   └── register.html
│       ├── dashboard/
│       │   ├── index.html           # Lista de cargas
│       │   ├── create.html          # Crear carga
│       │   ├── edit.html            # Editar carga
│       │   ├── show.html            # Detalle de carga
│       │   └── 404.html
│       ├── direcciones/
│       ├── empresas/
│       ├── choferes/
│       ├── publicadores/
│       ├── facturas/
│       ├── profile/
│       ├── admin/
│       └── home.html
│
├── bootstrap/
│   └── app.go                       # Boot de Goravel (facades, providers)
│
├── config/                          # Configuración (DB, cache, session…)
│
├── database/
│   └── migrations/                  # Migraciones GORM
│       ├── *_create_direcciones_table.go
│       ├── *_create_empresas_table.go
│       ├── *_create_users_table.go
│       ├── *_create_chofer_table.go
│       ├── *_create_publicador_table.go
│       ├── *_create_cargas_table.go
│       ├── *_create_facturas_table.go
│       └── *_create_cargas_historial_table.go
│
├── public/
│   ├── css/
│   │   └── style.css                # Design System BUFAL v7 (~2400 líneas)
│   ├── js/                          # Scripts adicionales
│   └── img/
│
├── routes/
│   └── web.go                       # Registro de todas las rutas
│
└── tests/                           # Helpers de test Go
    ├── helpers.go
    └── ...
```

---

## 🗃 Modelos de datos

### `User`

```go
type User struct {
    orm.Model                                    // ID, CreatedAt, UpdatedAt, DeletedAt
    Name, Email, Password string
    Role                    string              // admin | publicador | chofer

    // Contacto
    Phone, PhoneAlt, WhatsApp, Telegram *string
    EmergencyName, EmergencyPhone       *string

    // Empresa (opcional)
    EmpresaID *uint
    Empresa   *Empresa

    // Vínculo al perfil de rol (exactamente uno)
    PublicadorID *uint
    Publicador   *Publicador
    ChoferID     *uint
    Chofer       *Chofer

    // Ubicación
    Address, City, State, Country, PostalCode string
    Latitude, Longitude                        float64
    Radius                                     int

    // Preferencias (chofer)
    PreferredEquipmentTypes, PreferredCargoTypes string
    MaxWeight, MaxDistance                        float64
    PreferredRoutes                               string

    // Disponibilidad
    AvailableFrom, AvailableTo *time.Time
    Notes                      string

    IsActive  bool
    LastLogin *time.Time
    CreatedBy, UpdatedBy *uint
}
```

### `Empresa`

```go
type Empresa struct {
    ID              uint
    Tipo            TipoEmpresa   // broker | carrier | shipper | factoring | mixto
    NombreLegal     string
    NombreComercial *string
    TaxID           *string
    MCNumber        *string
    DOTNumber       *string
    DireccionID     *uint
    Telefono        *string
    Email           *string
    SitioWeb        *string
    CreditScore     *int
    DaysToPay       *float64
    OwnerID         *uint        // user que la creó
    Estado          EstadoEmpresa
    CreatedAt       time.Time
    UpdatedAt       *time.Time
    DeletedAt       *time.Time

    // Relaciones
    Direccion *Direccion
    Choferes  []Chofer
}
```

### `Chofer`

```go
type Chofer struct {
    ID         uint
    UserID     uint      // 1:1 con User
    EmpresaID  *uint     // opcional
    NumeroLicencia  string
    TipoLicencia    string
    PaisEmisionLicencia string
    FechaVencimientoLicencia *time.Time
    AniosExperiencia int
    TiposEquipoPermitidos string
    Certificaciones       string
    NumeroSeguro          string
    FechaVencimientoSeguro *time.Time
    Estado  EstadoChofer // disponible | en_viaje | inactivo
}
```

### `Publicador`

```go
type Publicador struct {
    ID         uint
    UserID     uint      // 1:1 con User
    EmpresaID  *uint     // opcional
    NumeroLicenciaBroker string
    PaisEmisionLicencia  string
    FechaVencimientoLicencia *time.Time
    AniosExperiencia int
    Especialidad     string
    Comision         float64
    CreditScore      int
    Estado  EstadoPublicador // activo | inactivo | suspendido
}
```

### `Carga`

```go
type Carga struct {
    ID                 uint
    NumeroReferencia   string

    // Actores
    PublicadorID uint      // quién publica
    EmpresaID    uint      // empresa del publicador
    ChoferID     *uint     // asignado al aceptar

    // Ruta
    OrigenDireccionID  uint
    DestinoDireccionID uint

    // Fechas
    FechaRecogida time.Time
    FechaEntrega  *time.Time

    // Carga
    TipoCarga   TipoCarga    // FTL | LTL
    TipoEquipo  TipoEquipo   // dry_van, reefer, flatbed, …
    PesoKg      *float64
    Commodity   *string

    // Distancias
    DistanciaKm     float64
    DistanciaRealKm *float64

    // Tarifa
    TarifaTotal *float64
    TarifaPorKm *float64
    Moneda      Moneda     // CUP | MLC | USD | EUR

    // Estado
    Estado    EstadoCarga // publicada, negociando, asignada, en_transito, entregada, cancelada
    Audiencia Audiencia   // load_board, red_privada, red_extendida

    // Relaciones
    Publicador       *Publicador
    Empresa          *Empresa
    Chofer           *Chofer
    OrigenDireccion  *Direccion
    DestinoDireccion *Direccion
    Factura          *Factura
}
```

### `Direccion`

```go
type Direccion struct {
    ID              uint
    Calle           *string
    Ciudad          string
    EstadoProvincia string
    CodigoPostal    *string
    Pais            string
    Latitud         *float64
    Longitud        *float64
    OwnerID         *uint
    CreatedAt       time.Time
    UpdatedAt       *time.Time
}
```

### `Factura`

```go
type Factura struct {
    ID         uint
    CargaID    uint
    EmisorID   uint
    ReceptorID uint
    ChoferID   *uint
    NumeroFactura string
    FechaEmision  time.Time
    FechaVencimiento *time.Time
    DistanciaKm  float64
    TarifaPorKm  float64
    Subtotal     float64
    Impuestos    float64
    Total        float64
    Moneda       Moneda
    Estado       EstadoFactura
    MetodoPago   *string
    FechaPago    *time.Time
}
```

---

## 🔐 Roles y permisos

| Rol | Descripción | Puede hacer |
|---|---|---|
| **admin** | Administrador del sistema | Todo |
| **publicador** | Broker que publica cargas | Crear/editar/eliminar **sus** cargas, **sus** direcciones, **su** empresa, facturas |
| **chofer** | Transportista que acepta cargas | Ver cargas publicadas, aceptar, editar **su** perfil, **sus** direcciones, **su** empresa |

### Reglas de ownership

| Recurso | Regla |
|---|---|
| `Empresa` | `Empresa.OwnerID == user.ID` o admin |
| `Chofer` | `Chofer.UserID == user.ID` o admin |
| `Publicador` | `Publicador.UserID == user.ID` o admin |
| `Carga` | `Carga.PublicadorID == user.Publicador.ID` o admin |
| `Direccion` | `Direccion.OwnerID == user.ID` o admin |

### Empresa por rol

- **Publicador** → solo puede crear/editar empresas tipo **`broker`**
- **Chofer** → solo puede crear/editar empresas tipo **`carrier`**

---

## 🛣 Rutas principales

### Públicas

| Método | Ruta | Handler |
|---|---|---|
| `GET` | `/login` | `AuthController.ShowLogin` |
| `POST` | `/login` | `AuthController.HandleLogin` |
| `GET` | `/register` | `AuthController.ShowRegister` |
| `POST` | `/register` | `AuthController.HandleRegister` |

### Protegidas (SessionAuth)

| Método | Ruta | Handler |
|---|---|---|
| `GET` | `/home` | `AuthController.ShowHome` |
| `GET` | `/logout` | `AuthController.Logout` |
| `GET` | `/profile` | `UserController.Show` |
| `GET` | `/profile/edit` | `UserController.Edit` |
| `POST` | `/profile/update` | `UserController.Update` |
| `GET` | `/loads` | `CargaController.Index` |
| `GET` | `/loads/:id<int>` | `CargaController.Show` |
| `GET` | `/direcciones` | `DireccionController.Index` |
| `GET` | `/empresas` | `EmpresaController.Index` |
| `GET` | `/choferes` | `ChoferController.Index` |
| `GET` | `/publicadores` | `PublicadorController.Index` |

### Escritura de cargas (PublicadorAuth)

| Método | Ruta | Handler |
|---|---|---|
| `GET` | `/loads/create` | `CargaController.Create` |
| `POST` | `/loads` | `CargaController.Store` |
| `GET` | `/loads/:id<int>/edit` | `CargaController.Edit` |
| `POST` | `/loads/:id<int>` | `CargaController.Update` |
| `POST` | `/loads/:id<int>/delete` | `CargaController.Delete` |
| `POST` | `/loads/:id<int>/assign-chofer` | `CargaController.AssignChofer` |

### Aceptar carga (ChoferAuth)

| Método | Ruta | Handler |
|---|---|---|
| `POST` | `/loads/:id<int>/accept` | `CargaController.AcceptLoad` |
| `POST` | `/loads/:id<int>/interest` | `LoadInterestController.SendInterest` |

### Shared (publicador + chofer + admin)

- `/direcciones/create`, `/direcciones`, `/direcciones/:id<int>/edit`, `/direcciones/:id<int>`, `/direcciones/:id<int>/delete`
- `/empresas/create`, `/empresas`, `/empresas/:id<int>/edit`, `/empresas/:id<int>`, `/empresas/:id<int>/delete`

### Admin

- `/admin/users`, `/admin/users/create`, `/admin/users/:id<int>/edit`, `/admin/users/:id<int>/toggle`, etc.

> ⚠️ **Importante**: todos los `:id` usan `<int>` para que Fiber no capture strings como `create` al matchear la ruta `:id`.

---

## 🎨 Frontend — Design System BUFAL

El frontend completo se apoya en un **design system propio** (~2400 líneas de CSS) que da coherencia visual a todas las vistas. Está documentado en `public/css/style.css`.

### Identidad visual

| Elemento | Valor | Uso |
|---|---|---|
| **Ámbar BUFAL** | `#F59E0B` | Acento principal, botones primarios, iconos activos |
| **Rojo** | `#EF4444` | Solo para danger (eliminar, cancelar) |
| **Verde** | `#10B981` | Success (entregada, aceptada) |
| **Negro búfalo** | `#0B0F14` | Texto principal, dark mode base |
| **Gris acero** | `#F3F4F6` → `#4B5563` | Superficies, bordes, texto secundario |

### Tipografía

- **Display**: `'TT Autonomous', 'Inter', system-ui` — títulos
- **Body**: `'TT Autonomous', 'Inter', system-ui` — cuerpo
- **Mono**: `'JetBrains Mono', ui-monospace` — IDs, coordenadas, montos

Tamaños: `xs` (0.75rem) → `3xl` (2.25rem).

### Componentes principales

| Componente | Clase base | Uso |
|---|---|---|
| Botón primario | `.btn.btn-primary` | Publicar carga, aceptar |
| Botón secundario | `.btn.btn-secondary` | Cancelar, volver |
| Botón danger | `.btn.btn-danger` | Eliminar |
| Botón outline | `.btn-outline-*` | Acciones secundarias |
| Card | `.card` + `.card-header` + `.card-body` | Contenedores de contenido |
| Alerta | `.alert.alert-success` / `.alert-danger` | Flash messages |
| Badge | `.badge.bg-success` / `.bg-warning` / `.bg-danger` | Estados |
| Tabla | `.tf-table` | Listados |
| Empty state | `.tf-empty` | Cuando no hay resultados |
| Form page | `.tf-form-page` | Layout de formularios con rail lateral |
| Form block | `.tf-form-block` | Sección de formulario |
| Route | `.tf-route` | A→B (origen/destino) |
| Progress | `.tf-progress` | Checklist lateral |
| Map picker | `.tf-map-picker` | Selector de coordenadas |
| Map pin | `.tf-map-pin--pickup` / `--dropoff` | Marcadores Leaflet |

### Páginas clave

| Página | Ruta | Clases raíz |
|---|---|---|
| Login | `/login` | `.tf-login` |
| Register | `/register` | `.tf-form-page` |
| Home | `/home` | `.tf-home` + `.bf-app` |
| Lista cargas | `/loads` | `.tf-loads-page` |
| Detalle carga | `/loads/:id` | `.tf-profile` |
| Editar carga | `/loads/:id/edit` | `.tf-form-page` |
| Perfil | `/profile` | `.tf-profile` |

### App Shell (vista autenticada)

El layout base usa un **shell tipo WordPress**:

```
┌──────────────────────────────────────────────────┐
│  TOPBAR (64px)                                   │
│  [User Menu]      [Topnav]       [Theme + Brand] │
├────────────┬─────────────────────────────────────┤
│            │                                     │
│  SIDEBAR   │            MAIN                     │
│  (260px)   │        (max 1400px)                  │
│            │                                      │
│  - Cargas  │   <h1>Título de página</h1>          │
│  - Empresas│                                       │
│  - Choferes│   [Contenido / tarjetas / tablas]    │
│  - Perfil  │                                       │
│            │                                       │
└────────────┴─────────────────────────────────────┘
```

Clases:
- `.bf-app` — wrapper general
- `.bf-topbar` — barra superior con grid `1fr auto 1fr`
- `.bf-user-btn` — botón de usuario (dropdown)
- `.bf-topnav` — navegación central
- `.bf-sidebar` — barra lateral con `.bf-sidebar__link` (activo con `aria-current="page"`)
- `.bf-main` — contenido principal

---

## 🧬 Arquitectura del CSS

El CSS sigue una **metodología de design tokens + BEM + utilidades**. Está dividido en 33 secciones numeradas.

### 1. Design tokens (CSS Custom Properties)

Todos los valores visuales están centralizados en custom properties en `:root`:

```css
:root {
  /* Colores */
  --tf-yellow-500: #F59E0B;
  --tf-red-500:    #EF4444;
  --tf-success:    #10B981;
  --tf-gray-050:   #F9FAFB;
  ...
  /* Tipografía */
  --tf-font-display: 'TT Autonomous', 'Inter', system-ui;
  --tf-fs-xs: 0.75rem;
  --tf-fs-3xl: 2.25rem;
  --tf-fw-semibold: 600;
  /* Espaciado */
  --tf-space-1: 0.25rem;
  --tf-space-12: 3rem;
  /* Radios */
  --tf-radius-sm: 6px;
  --tf-radius-pill: 999px;
  /* Sombras */
  --tf-shadow-sm: 0 1px 2px rgba(11,15,20,.06), 0 1px 3px rgba(11,15,20,.04);
  --tf-shadow-red: 0 6px 18px rgba(245,158,11,.28);
  /* Transiciones */
  --tf-dur-fast: 140ms;
  --tf-ease: cubic-bezier(.22,1,.36,1);
  /* Gradientes */
  --tf-gradient-red: linear-gradient(135deg, #FBBF24 0%, #F59E0B 55%, #B45309 100%);
}
```

Ventaja: cambiar un token afecta todo el sistema. Ejemplo: cambiar `--tf-yellow-500` a morado transforma la identidad visual completa.

### 2. Tema claro/oscuro

```css
:root { /* tokens del tema claro */ }
[data-theme="dark"] { /* overrides del tema oscuro */ }
```

El toggle en el topbar cambia `document.documentElement[data-theme]` y persiste en `localStorage` bajo `tf-theme`. Un script en `<head>` aplica el tema **antes** del primer paint para evitar flash.

```html
<script>
  (function () {
    var saved = localStorage.getItem('tf-theme') || 'light';
    document.documentElement.setAttribute('data-theme', saved);
  })();
</script>
```

### 3. Estructura de secciones del CSS

| # | Sección | Contenido |
|---|---|---|
| 1 | Fuentes | `@font-face` (TT Autonomous) |
| 2 | Tokens light | `:root` con todas las variables |
| 3 | Tokens dark | `[data-theme="dark"]` overrides |
| 4 | Base / reset | `*, body, html, ::selection` |
| 5 | Layout | `.tf-container` |
| 6 | Cards | `.card`, `.card-header`, `.card-body` |
| 7 | Section title | `.section-title` |
| 8 | Form controls | `.form-control`, `.form-select`, `.form-label` |
| 9 | Buttons | `.btn`, `.btn-primary`, `.btn-secondary`, …, `.btn-outline-*` |
| 10 | Badges | `.badge.bg-*` |
| 11 | Alerts | `.alert`, `.alert-success`, `.alert-danger` |
| 12 | Tables | `.table`, `.table-hover` |
| 13 | Pagination | `.pagination`, `.page-link` |
| 14 | Componentes custom | `.address-block`, `.field-label`, `.rate-amount` |
| 15 | Dashboard shell | `.tf-shell`, `.tf-sidebar`, `.tf-topbar` |
| 16 | Login | `.tf-login`, `.tf-login-card` |
| 17 | Animaciones | `@keyframes tf-fade-in`, `tf-slide-down`, `tf-scale-in` |
| 18 | Responsive base | `@media (max-width: …)` |
| 19 | Utilidades | `.tf-mt-*`, `.tf-grid--*`, `.tf-col-*`, `.tf-flex--*` |
| 20 | Page header | `.tf-page-header` |
| 21 | Form page | `.tf-form-page`, `.tf-form-block`, `.tf-rail-card`, `.tf-progress`, `.tf-route`, `.tf-summary` |
| 22 | Loads page | `.tf-loads-page`, `.tf-filters`, `.tf-table-card`, `.tf-empty`, `.tf-pagination-wrap` |
| 23 | Home v2 | `.tf-home`, `.tf-quicknav`, `.tf-user-card`, `.tf-home-stats`, `.tf-action-tile`, `.tf-avail` |
| 24 | Profile view | `.tf-profile`, `.tf-info-block`, `.tf-info-list`, `.tf-chip` |
| 25 | Admin crear usuario | `.tf-password-meter`, `.tf-role-grid`, `.tf-user-preview` |
| 26 | Edit load | `.tf-status-selector`, `.tf-rail-card--load`, `.tf-preview__status` |
| 27 | Register | `.tf-login-card--register` |
| 28 | Accesibilidad / print | `@media (prefers-reduced-motion)`, `@media print` |
| 29 | Destinations | `.tf-pagination__info` |
| 30 | Load map | `#route-map`, `.tf-map-pin`, Leaflet overrides |
| 31 | Map picker | `.tf-map-picker__*` |
| 32 | Role selector | `.tf-role-selector`, `.tf-role-option` |
| 33 | Empresa mode | `.tf-empresa-mode`, `.tf-mode-option` |
| — | App shell | `.bf-app`, `.bf-topbar`, `.bf-sidebar`, `.bf-main` |

### 4. Convenciones de nombres

- **Prefijo `tf-`** → "TruckFast" / sistema base (formularios, tablas, cards).
- **Prefijo `bf-`** → "Búfalo" / App Shell (topbar, sidebar, main).
- **BEM**: `.block__element--modifier` → `.tf-form-block__head`, `.tf-role-option--selected`.
- **Estados**: `.is-active`, `.is-selected`, `.is-complete`, `.is-current`, `.is-collapsed`.
- **Utilidades**: `.tf-text-center`, `.tf-mt-4`, `.tf-w-full`, `.tf-col-6`.

### 5. Responsive

Breakpoints principales:

| Breakpoint | Objetivo |
|---|---|
| `max-width: 1100px` | Form page pasa a 1 columna |
| `max-width: 991.98px` | Sidebar se colapsa a tabs horizontales |
| `max-width: 767.98px` | Grids de 3-4 cols pasan a 1-2 |
| `max-width: 575.98px` | Mobile: padding reducido, tipografía escalada |

Ejemplo:
```css
@media (max-width: 991.98px) {
  .bf-body { grid-template-columns: 1fr; }
  .bf-sidebar {
    position: sticky;
    top: 64px;
    height: auto;
    overflow-x: auto;
  }
  .bf-sidebar__nav { flex-direction: row; overflow-x: auto; }
  .bf-sidebar__label { display: none; }
}
```

### 6. Componentes especiales

#### Botones industriales

Los botones tienen un **estilo industrial** característico:
- Una barra de color superior (`.btn::before`) que da sensación metálica.
- Un shine diagonal al hover (`.btn::after`).
- Elevación al hover (`transform: translateY(-2px)`).
- Color contrastante en los iconos.

```css
.btn::before {
  content: "";
  position: absolute;
  inset: 0 0 auto 0;
  height: 3px;
  background: currentColor;
  opacity: .55;
}
.btn:hover::after {
  transform: translateX(100%); /* shine effect */
}
```

#### Route (A → B)

Grid de 3 columnas: punto A · conector · punto B.

```html
<div class="tf-route">
  <div class="tf-route__point tf-route__point--pickup">…</div>
  <div class="tf-route__connector"><i class="bi bi-arrow-right"></i></div>
  <div class="tf-route__point tf-route__point--dropoff">…</div>
</div>
```

#### Progress checklist

```html
<ol class="tf-progress">
  <li class="tf-progress__step is-complete">…</li>
  <li class="tf-progress__step is-current">…</li>
  <li class="tf-progress__step">…</li>
</ol>
```

Estados:
- `.is-current` — paso actual (ámbar)
- `.is-complete` — paso completado (verde, checkmark)

El JS recalcula dinámicamente en cada `input` / `change`.

#### Map picker

Integra **Leaflet + Nominatim**:
- Búsqueda con Nominatim (restringida a Cuba con `countrycodes=cu`).
- Clic o drag del marcador para seleccionar coordenadas.
- Reverse geocoding → autocompleta `ciudad`, `estado_provincia`, `pais`, `codigo_postal`.
- Sincronización bidireccional con inputs de lat/lng.

```html
<div class="tf-map-picker" data-map-picker>
  <div class="tf-map-picker__search">…</div>
  <div class="tf-map-picker__map" data-map></div>
  <div class="tf-map-picker__footer">
    <span class="tf-map-picker__coords" data-coords></span>
    <span class="tf-map-picker__hint" data-address-hint></span>
  </div>
</div>
```

> El mapa **solo se inicializa cuando el contenedor es visible** (`offsetWidth > 0`). Esto evita problemas con Leaflet en contenedores `display: none`.

### 7. Accesibilidad

- **Focus visible**: `:focus-visible` con outline ámbar.
- **Contraste AA**: ámbar sobre negro, blanco sobre ámbar oscuro.
- **`aria-current="page"`** para marcar el link activo.
- **`aria-expanded`** en el botón de usuario (dropdown).
- **`prefers-reduced-motion`**: desactiva animaciones para usuarios que lo requieran.
- **`@media print`**: oculta topbar, sidebar, acciones; hace el contenido imprimible.

### 8. Cómo extender el sistema

#### Añadir un nuevo color

```css
:root {
  --tf-purple-500: #8B5CF6;
}
[data-theme="dark"] {
  --tf-purple-500: #A78BFA;
}
```

Y usarlo:
```css
.badge.bg-purple { background: var(--tf-purple-500); }
```

#### Añadir un nuevo componente

```css
/* 34. MI COMPONENTE
   ------------------------------------------------------------ */
.tf-mi-componente {
  /* Usar tokens, no valores hardcoded */
  background: var(--tf-surface);
  border: 1px solid var(--tf-border);
  border-radius: var(--tf-radius-md);
  padding: var(--tf-space-4);
  box-shadow: var(--tf-shadow-sm);
  transition: all var(--tf-dur) var(--tf-ease);
}
.tf-mi-componente:hover {
  border-color: var(--tf-yellow-500);
  transform: translateY(-2px);
}
```

#### Añadir un tema

```css
[data-theme="midnight"] {
  --tf-bg: #0A0A1A;
  --tf-surface: #12122A;
  --tf-text: #E0E0FF;
  /* … */
}
```

Y el toggle cambia a `midnight`.

---

## 🚀 Instalación y ejecución

### Requisitos

- Go 1.22+
- PostgreSQL 14+
- Redis 6+

### Configuración

Crea un `.env`:

```env
APP_ENV=local
APP_KEY=base64:...

DB_HOST=127.0.0.1
DB_PORT=5432
DB_DATABASE=datclone
DB_USERNAME=postgres
DB_PASSWORD=postgres

REDIS_HOST=127.0.0.1
REDIS_PORT=6379
REDIS_DB=0
REDIS_POOL_SIZE=10
```

### Migrar y arrancar

```bash
# 1. Migrar
go run . artisan migrate:fresh

# 2. Levantar el servidor
go run .

# → http://localhost:3000
```

Al arrancar, si no existe el admin `admin@example.com` / `Admin123!` se crea automáticamente.

### Flujos disponibles

1. **Registrar** un publicador o chofer en `/register`.
2. **Login** en `/login`.
3. **Home** muestra dashboard según rol.
4. **Crear dirección** con mapa interactivo.
5. **Crear carga** (solo publicador).
6. **Aceptar carga** (solo chofer).
7. **Editar/eliminar** propios recursos.

---

## 🧪 Testing

### Tests automatizados Go

```bash
# Tests de servicios
go test ./app/services/ -v

# Tests de controllers
go test ./app/http/controllers/ -v

# Cobertura
go test ./... -cover
```

### Scripts de test end-to-end

Los scripts usan `curl` + `grep` para simular flujos completos:

```bash
# 1. Reset de BD
go run . artisan migrate:fresh
go run .

# 2. En otra terminal
./test_register_modes.sh      # 3 modos de registro + validaciones
./test_ownership.sh           # Ownership entre usuarios
./test_cargas_crud.sh         # CRUD completo de cargas
```

Cada script imprime `OK` / `FAIL` por cada caso. Al final muestra un resumen con los IDs creados.

### Casos cubiertos

| Test | Cubre |
|---|---|
| `test_register_modes.sh` | Registro con `empresa_mode=new`, `existing`, `none`; validación de email duplicado, password débil, rol inválido |
| `test_ownership.sh` | Edición de dirección/empresa/carga por no-dueño → rechazado |
| `test_cargas_crud.sh` | Ciclo completo: crear, listar, editar, aceptar, ver detalle, eliminar; ownership; accept por chofer |

---

## 📝 Notas de diseño

### Por qué `<int>` en las rutas

Fiber v3 permite tipar los parámetros de ruta:

```go
app.Get("/loads/:id<int>", ctrl.Show)
```

Sin `<int>`, `/loads/create` matchearía `/loads/:id` con `id="create"`. Con `<int>`, Fiber exige que el valor sea entero y salta a la siguiente ruta.

### Por qué `strPtr` en helpers

Los requests usan `string` para campos opcionales (porque el form envía `""` cuando están vacíos). El modelo usa `*string` para mapear `NULL` en PostgreSQL. `strPtr` traduce entre ambos:

```go
func strPtr(s string) *string {
    if s == "" {
        return nil
    }
    return &s
}
```

### Por qué `defer` en JS del mapa

Leaflet no puede calcular el tamaño de un contenedor `display: none` (`offsetWidth == 0`). Por eso el mapa se inicializa **después** de que el bloque sea visible:

```js
waitForVisible(el, () => window.__initRegisterMapPicker());
```

### Por qué el CSS tiene prefijos `tf-` y `bf-`

- `tf-` = sistema base (TruckFast): formularios, tablas, cards, botones.
- `bf-` = app shell (Búfalo): topbar, sidebar, layout principal.

Esto permite reutilizar el design system en otras páginas sin depender del shell.

---

## 📄 Licencia

Proyecto privado. Todos los derechos reservados.

---

## 👥 Autor

Desarrollado por **p3terpl4y** — 2026.
