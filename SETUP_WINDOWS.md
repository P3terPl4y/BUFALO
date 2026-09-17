# Puesta en marcha local en Windows

Esta guía deja el proyecto en `D:\xampp2\BUFALO`. XAMPP aporta Apache y MySQL/MariaDB, pero BUFALO usa PostgreSQL y Redis; Apache no ejecuta la aplicación Go.

## Requisitos

- Go 1.22 o superior.
- PostgreSQL 14 o superior.
- Redis 6 o superior.

Redis puede ejecutarse desde la instalación local disponible en Laragon. PostgreSQL debe estar instalado como servicio de Windows o ejecutarse desde una instalación existente.

## Dependencias del proyecto

Desde PowerShell:

```powershell
Set-Location D:\xampp2\BUFALO
go mod download
go test ./...
```

## Variables de entorno

Copia el ejemplo y ajusta la contraseña de PostgreSQL:

```powershell
Copy-Item .env.example .env
```

Si el repositorio no trae `.env.example`, crea `.env` con al menos:

```env
APP_ENV=local
APP_KEY=0123456789abcdef0123456789abcdef
DB_CONNECTION=postgres
DB_HOST=127.0.0.1
DB_PORT=5432
DB_DATABASE=bufalo
DB_USERNAME=postgres
DB_PASSWORD=postgres
DB_SCHEMA=public
DB_SSLMODE=disable
REDIS_HOST=127.0.0.1
REDIS_PORT=6379
REDIS_DB=0
REDIS_POOL_SIZE=10
```

Antes de migrar, crea la base de datos `bufalo` en PostgreSQL. Después arranca Redis y ejecuta:

```powershell
go run . artisan migrate:fresh
go run .
```

La aplicación queda disponible en `http://localhost:3000`. La cuenta administrativa inicial se documenta en `README.md`; cambia sus credenciales en cuanto accedas.

## Rama de trabajo

Los cambios de esta revisión se realizan en `fix-main-yoenis`. Para comprobarlo:

```powershell
git status --short --branch
```
