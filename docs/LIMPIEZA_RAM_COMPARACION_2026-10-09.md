# RAM, limpieza de datos residuales y comparación con commits — 9 de octubre de 2026

## Consumo medido

La muestra final se tomó después de verificar login y logout. PSS reparte la memoria compartida: sumar el RSS de los procesos de PostgreSQL la contaría repetidamente. Las cifras corresponden a la instancia local, no a producción.

| Componente | RSS sumado, MiB | PSS, MiB | Estado |
|---|---:|---:|---|
| BUFALO HTTP | 54.03 | 54.02 | Activo local |
| PostgreSQL BUFALO (cluster y conexiones) | 172.18 | 80.85 | Activo local |
| Redis BUFALO | 7.36 | 3.44 | Activo local |
| SMTP de pruebas BUFALO | 12.88 | 7.12 | Activo local |

**Total proporcional de los componentes dedicados: 145.43 MiB.** Los servicios de producción y govulncheck están apagados; no aportan procesos residentes actualmente. El capturador SMTP de pruebas se conserva porque la instancia local está configurada para usarlo y retirarlo haría fallar sus envíos de interés.

PostgreSQL 5432, Redis 6379 y cloudflared son compartidos con otros proyectos y se excluyen de este total. El clúster PostgreSQL compartido también contiene boti_wsp, chita y halcon. La base histórica de producción `bufalo` no tenía conexiones y se conservó; borrar datos reales no forma parte de limpiar bases residuales de pruebas.

## Limpieza realizada

Se verificó que las bases candidatas no tuvieran conexiones; no se forzó la terminación de clientes ni se usó DROP DATABASE FORCE.

- PostgreSQL `127.0.0.1:55448`: eliminada `bufalo_auth_race_test`.
- PostgreSQL `127.0.0.1:55448`: eliminada `bufalo_auth_utf8_test`.
- PostgreSQL `127.0.0.1:55448`: eliminada `bufalo_restore_validation`.
- PostgreSQL `127.0.0.1:55448`: eliminada `bufalo_security_test`.
- PostgreSQL compartido `127.0.0.1:5432`: eliminada `bufalo_test`, sin tablas de aplicación ni conexiones.
- Espacio lógico de bases eliminado: **43.63 MiB**. No incluye un cálculo del tamaño total del directorio del clúster ni de WAL.
- Redis dedicado `127.0.0.1:56348`, DB 1: eliminadas 8 claves temporales presentes al ejecutar la limpieza. Una clave de las nueve inventariadas previamente venció antes de la operación. Se ejecutó MEMORY PURGE.
- Las claves generadas por la comprobación posterior también se retiraron: DB 1 terminó con cero claves. No se ejecutó FLUSHALL. DB 0 ya estaba vacía.
- Redis compartido 6379 no se modificó. Sus credenciales disponibles para el proyecto no permitieron inventariarlo; no se afirma que esté libre de residuos.
- Se conservan `bufalo_local_test`, necesaria para la aplicación local, y las bases de sistema postgres/template0/template1.

La limpieza libera almacenamiento, pero no vacía automáticamente los buffers/cachés de PostgreSQL. La muestra inmediatamente posterior apenas cambió (PSS agregado 142,18 → 142,71 MiB); la posterior al smoke llegó a 145.43 MiB por trabajo adicional. No hay una reducción demostrada de RAM del proceso debida a eliminar estas bases. No se reinició el clúster activo solo para reducir la cifra.

## Validación posterior

La instancia siguió en 33333. Salud y disponibilidad internas devolvieron 200; login y registro GET, login de cuenta ficticia, inicio autenticado, rechazo CSRF, logout POST e invalidación de sesión pasaron. No se enviaron correos reales. Registro: `storage/implementation-backup/cleanup-smoke-20261009.json`.

Las bases de suites se eliminaron deliberadamente: antes de volver a ejecutarlas habrá que recrear una base aislada UTF-8 y aplicar sus migraciones. La suite anterior aprobada no se volvió a ejecutar durante esta limpieza; hacerlo recrearía los residuos que se acaba de retirar. Ejemplo de preparación:

```sh
createdb -h 127.0.0.1 -p 55448 -U peter -T template0 -E UTF8 bufalo_auth_utf8_test
```

## Comparación objetiva con el historial

Se inspeccionaron `231b607` (3 de octubre), `a04ac81` (5 de octubre, métricas/exportaciones) y `ddeea6f` (5 de octubre, HEAD). Las correcciones actuales están en el árbol de trabajo y aún no están consolidadas en commits: clonar HEAD no obtiene la versión validada. No se alteraron ni descartaron cambios existentes.

| Área | Último commit, ddeea6f | Árbol actual validado |
|---|---|---|
| Credenciales | Cambio directo de correo/contraseña desde sesión; sin huella para revocar sesiones por cambio | Contraseña actual, correo confirmado, huella de credenciales y revocación |
| Lectura y edición de cargas | Inicio sin filtro de audiencia; paginación sin normalizar; actualización genérica sin estado | Filtros por actor/red, tope 100, direcciones autorizadas y protección de operaciones cerradas |
| Interés en cargas | Destinatario y estado tomados del formulario, HTML interpolado | Modal, destinatario desde BD, escape HTML, cuotas y deduplicación |
| Concurrencia | Agregados de calificaciones calculados sin bloqueo previo del chofer | Bloqueo antes de insertar/calcular; regresión concurrente PostgreSQL aprobada |
| Abuso y recursos | Sin los actuales módulos de admisión y guardia de tráfico | Admisión, límites de cuerpos/concurrencia/IP, sesiones/cuotas acotadas y visibilidad de rechazos |
| Dependencias/arranque | Proveedores AI/OpenAI/gRPC/telemetría cargados | Proveedores no utilizados retirados y binario estático optimizado |

La versión actual tiene 200 resultados aprobados, incluidos subtests (127 pruebas de nivel superior en 18 paquetes), análisis estático aprobado y pruebas enfocadas con detector de carreras aprobadas. No se ejecutaron esas mismas pruebas sobre cada commit antiguo, por lo que no se atribuye una tasa de éxito comparable al historial.

El binario antiguo inventariado en la auditoría pesaba 109.654.653 bytes; el candidato actual 58.982.562 bytes: **46,2 % menos tamaño** (104,57 → 56,25 MiB). Las recetas de compilación también difieren: parte de la reducción procede de quitar símbolos, además de reducir dependencias. Es una mejora de distribución/disco, no una reducción equivalente de RAM.

Las mediciones locales históricas del 8 de octubre muestran mejoras en GET de formularios al evitar recargar plantillas. Son pruebas de configuraciones y árboles de trabajo, no benchmarks reproducidos por commit. Además, producción ya desactivaba la recarga: esas cifras no demuestran que la causa histórica de las caídas fuera esa ni permiten afirmar una ganancia porcentual de rendimiento en producción.

## Valoración y pendientes

BUFALO está sustancialmente mejor en seguridad funcional y control de recursos que los commits inspeccionados. El candidato funciona en los flujos ensayados. No se considera cerrado para producción: las mejoras aún requieren consolidación en Git y CI, validación del despliegue completo y observación con carga representativa. Producción permanece retirada.

También quedan el reenvío controlado de confirmaciones, una outbox para recuperar notificaciones si el proceso cae después de reservar el envío, inventario/migración de FK y CHECK, el flujo aprobado de asociación a empresas y dividir AdminController. Govulncheck no encontró vulnerabilidades alcanzables, pero conserva un aviso en un módulo requerido cuyo código vulnerable no aparece llamado. Las pruebas no demuestran resistencia a millones de solicitudes/s.

## Evidencia

- `storage/implementation-backup/cleanup-memory-before.json` y `cleanup-memory-after.json`.
- `storage/implementation-backup/database-cleanup-20261009.json`.
- `storage/implementation-backup/shared-postgres-inventory-20261009.json` (inventario previo a retirar bufalo_test).
- `storage/implementation-backup/cleanup-smoke-20261009.json`.
- `docs/ACTUALIZACIONES_2026-10-09.md` y `docs/OPTIMIZACION_LOCAL_2026-10-08.md`.
