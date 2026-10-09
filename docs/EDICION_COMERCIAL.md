# Edición comercial base

Esta rama conserva las correcciones de seguridad, estabilidad, administración y despliegue guardadas en el respaldo `backup/progreso-2026-10-09`. Deja fuera el laboratorio para diseñar facturas y las exportaciones del panel de salud solicitadas después; esas funciones continúan en `main`.

El precio de venta no se fija en el código. Antes de entregar esta edición a un comprador hay que ejecutar la batería de aceptación en su entorno, revisar licencias y dependencias, definir qué soporte y actualizaciones incluye la oferta y preparar credenciales propias del comprador. No se deben reutilizar datos, claves ni archivos locales de producción.

Para continuar cada línea:

- Edición comercial: `venta/edicion-base-2000-usd`
- Desarrollo con módulos nuevos: `main`
- Respaldo completo del punto de separación: `backup/progreso-2026-10-09` (commit `1fac614`)
