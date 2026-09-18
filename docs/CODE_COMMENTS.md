# Comentarios y documentación del código

El código debe explicar decisiones y contratos que no sean evidentes por el nombre de la función. Los comentarios se escriben en español porque el dominio y la interfaz del producto están en español; los nombres de paquetes, tipos y métodos siguen las convenciones de Go.

## Qué debe documentarse

- Cada tipo exportado debe tener un comentario que empiece por su nombre.
- Cada función o método exportado debe explicar su propósito, parámetros relevantes, resultado y efectos secundarios.
- Los controladores deben indicar el middleware requerido, el recurso que modifican y la redirección esperada.
- Los servicios deben documentar reglas de negocio, ownership, transacciones e idempotencia.
- Los modelos deben explicar relaciones, campos opcionales y estados del dominio; los tags GORM no necesitan una traducción campo por campo.
- Las migraciones deben indicar qué tablas crean, dependencias y cualquier decisión especial para Supabase.
- Los bloques de configuración deben explicar por qué se activa una opción cuando el valor no es obvio.

## Qué debe evitarse

- Comentarios que repitan literalmente el nombre: `// GetByID obtiene por ID` no aporta contexto.
- Comentarios desactualizados después de cambiar la implementación.
- Contraseñas, tokens, URLs privadas o datos de conexión reales.
- Comentarios sobre cada línea de una operación sencilla.

## Ejemplo para un servicio

```go
// GetPage devuelve una ventana ordenada de direcciones y el total sin filtrar.
// page es 1-based; perPage se normaliza a 10 cuando llega vacío o inválido.
// El total permite construir la paginación sin hacer una segunda consulta en
// el controlador.
func (s *DireccionService) GetPage(page, perPage int) ([]models.Direccion, int64, error) {
    // ...
}
```

## Ejemplo para un controlador

```go
// Store valida y persiste una dirección perteneciente al usuario autenticado.
// Ante un error vuelve al formulario con flash_error; tras crearla redirige al
// detalle para que el usuario pueda confirmar el resultado.
func (c *DireccionController) Store(ctx fiber.Ctx) error {
    // ...
}
```

## Revisión antes de un commit

```powershell
gofmt -w .
go test .
git diff --check
```

Cuando se agregue un método exportado, el comentario debe agregarse en el mismo commit. La guía general de arquitectura está en [`DEVELOPMENT.md`](DEVELOPMENT.md).
