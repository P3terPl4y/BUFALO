# Resultados de pruebas de servicios y controladores

Fecha de ejecución: 2026-10-09. Se revisaron estáticamente 29 archivos Go de
`app/services` y 32 de `app/http/controllers`, además de los requests, rutas y
pruebas relacionadas. Las suites directamente asociadas contienen 60 funciones
de prueba. Esta revisión incluye cobertura funcional y adversarial existente; no
equivale a probar todas las ramas y combinaciones de permiso de cada método.

## Resultado

No se encontraron fallos reproducibles en las pruebas ejecutadas. Las suites de
servicios y controladores, los recorridos web, redteam y las pasadas con detector
de carreras terminaron correctamente. Se añadieron dos regresiones concretas:

- Búsqueda de empresas con `%' OR 1=1 --`: la entrada se trató como texto,
  devolvió cero resultados y no alteró los dos registros de prueba.
- Formulario de carga con `empresa_id`, `publicador_id`, `chofer_id` y `estado`
  forjados: la carga quedó ligada al publicador/empresa autenticados, sin chofer
  asignado y en estado publicada.

## Validación ejecutada

| Suite | Resultado |
|---|---|
| `go test . ./app/billing ./app/community ./app/dbresilience ./app/exports ./app/models ./app/monitoring ./app/http/middleware ./app/viewhelpers ./routes -count=1` | Aprobada; middleware probó Redis/socket local real. `routes` no tiene pruebas propias. |
| `go test -p 1 -count=1 ./app/services` | Aprobada; 4,4 s en la primera ejecución y 4,2 s al incluir la nueva prueba de inyección. |
| `go test -p 1 -count=1 ./app/http/controllers` | Aprobada; 22,6 s. |
| `go test -p 1 -count=1 ./tests/feature` | Aprobada; 30,8 s. |
| `go test -p 1 -count=1 ./tests/redteam` | Aprobada; 92,4 s inicialmente y 95,7 s tras añadir la prueba de formulario manipulado. |
| `go test -race -p 1 -count=1 ./app/services` | Aprobada; sin carreras detectadas. |
| `go test -race -p 1 -count=1 ./app/http/controllers` | Aprobada; sin carreras detectadas. |
| `go vet ./...`, `go build ./...`, `git diff --check` | Aprobados después de añadir las regresiones. |

Las pruebas con base de datos usaron clúster PostgreSQL temporal en `/tmp`, con
bases `_test` independientes por suite. Se eliminó el clúster al terminar. No se
conectó ni envió tráfico de prueba al servicio o base de producción.

## Controles ejercitados

Las pruebas existentes comprobaron login y registro con CSRF, respuestas
genéricas de autenticación, rechazo de elevación de rol, confirmación de correo
de un solo uso y concurrencia; autorización entre usuarios y empresas; propiedad
de direcciones y cargas; rechazo de estados y facturas inválidos; formularios con
importes, fechas e IDs manipulados; asignación concurrente de una carga; límites
de tamaño y acceso del chat; validación de plantillas; notificaciones/outbox y
recuperación concurrente de entregas. Las pruebas de servicios también verifican
transacciones de creación/actualización y propagación de fallos de infraestructura.

La inspección estática encontró consultas con valores de usuario enlazados como
parámetros. El único `OrderByRaw` de producción usa una expresión constante; no
se encontró ordenación SQL construida desde texto del formulario. Los handlers
construyen mapas de actualización desde campos explícitos; la regresión de carga
comprueba la protección de ownership y estado ante campos adicionales forjados.

## Límites y trabajo pendiente

Esta ejecución no es fuzzing de todos los endpoints, un pentest externo ni una
prueba de carga/DDoS. No se ejecutó `govulncheck` porque no está instalado en el
entorno. Quedan por hacer una matriz automatizada de entradas hostiles para todos
los formularios/controladores, fuzzing de parsers y uploads, pruebas de caída y
recuperación de PostgreSQL/Redis por endpoint, revisión completa de configuración
de despliegue y análisis de dependencias. Los resultados acreditan los flujos
probados y las pruebas de regresión indicadas, no ausencia total de vulnerabilidades.
