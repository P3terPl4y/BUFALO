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

## Seguimiento de CI

El 2026-10-09 GitHub Actions falló en `tests/unit`. La prueba de correo esperaba
éxito aunque `EmailService` usa SMTP directamente y la variable `MAIL_MAILER=log`
no afecta ese servicio. En el entorno local `.env` proporcionaba `MAIL_HOST`; el
runner aislado no, así que el resultado dependía de la configuración de la máquina
y podía contactar un servidor SMTP real. Se cambió la prueba para exigir un fallo
seguro sin SMTP, se fija `MAIL_HOST` vacío en TestMain y CI, y la entrega válida se
cubre con el servidor SMTP local de `app/maildelivery`. La suite unitaria pasó en
un PostgreSQL temporal aislado.

Los escaneos de CI anteriores terminaron con códigos 137 y 143; luego, el escaneo
por paquetes de la versión anterior terminó con código 3 porque encontró avisos
reales. El análisis local identificó 13 avisos en Go 1.26.8 y `golang.org/x/net`
0.59.0, incluidos fallos de disponibilidad en HTTP y límites de memoria. Se
actualizaron el toolchain a Go 1.26.9 y `golang.org/x/net` a 0.60.0; el release
oficial de Go 1.26.9 incluye correcciones de seguridad en `net/http`, `crypto/tls`,
`net/textproto`, `html/template` y `os`.

Tras la actualización, `govulncheck -scan=package ./...` terminó con cero avisos:
pico 171.536.384 bytes y 33,1 s. El análisis completo de símbolos también pasó:
cero vulnerabilidades alcanzables, pico 1.505.312.768 bytes bajo el cgroup de
2 GiB, 75,1 s de CPU y 76,2 s de duración. Reportó un aviso en un módulo requerido
que no aparece importado ni alcanzado por el código; el chequeo de importaciones
contra OpenPGP también sigue activo en CI. Se limitó el workflow para compilar el
analizador dentro del cgroup, usar el modo por paquetes y mantener el máximo de
2 GiB. El build de Docker y el workflow con las dependencias parcheadas quedan
pendientes de validación remota.

Con Go 1.26.9 pasaron localmente `go vet`, build optimizado, pruebas de servicios,
controladores, journeys, redteam, unitarias, middleware con Redis, sesiones y
paquetes de soporte; las bases de datos fueron temporales y se detuvieron tras las
pruebas. La instancia de producción siguió respondiendo 200, con el mismo PID y
aprox. 20 MiB de RAM mientras el análisis de seguridad se ejecutaba. El binario
actual de producción aún debe reemplazarse por el candidato parcheado tras cerrar
la validación remota y preflight.
