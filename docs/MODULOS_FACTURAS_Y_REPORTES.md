# Módulos de informes y diseño de facturas

## Salud e informes

La vista `/admin/health` y las exportaciones parten de `healthSnapshot`. Esto mantiene iguales la lectura de servicios, contadores, cola de notificaciones, protección de tráfico, tendencias e historial. El exportador recibe secciones con encabezados y filas, y las presenta en Excel o CSV respetando el tab seleccionado y los filtros de estado, método y ruta.

PDF abre la impresión del navegador sobre la vista de salud existente. Así conserva la maquetación visible, sus tarjetas, gráficas y tabla; la hoja de impresión oculta la navegación y los controles. El usuario puede guardar esa vista como PDF desde el diálogo de impresión.

`app/exports.Section` es la unidad extensible para agrupar un informe. Los módulos pueden añadir secciones sin alterar los generadores CSV/XLSX/PDF. XLSX comparte los encabezados BUFALO y conserva la primera fila al desplazarse. CSV lleva BOM UTF-8 y neutraliza texto que Excel interpretaría como fórmula.

## Diseños de factura por broker

`/facturas/disenador` es una herramienta autenticada para publicadores. Guarda hasta 24 plantillas por perfil de broker en `factura_plantillas`; los diseños ya no dependen del navegador. Un publicador puede marcar una como predeterminada, seleccionarla al crear un borrador o cambiar el formato asociado a una factura existente. Cada factura conserva su selección, mientras los cambios al diseño guardado se reflejan en las facturas que lo usan.

La factura seleccionada se presenta con sus datos reales en la vista del chofer y puede imprimirse o guardarse como PDF. Los bloques de partes, datos y totales son obligatorios para no ocultar quién cobra, quién paga o el importe; el servidor valida formato, color, orden, duplicados, propiedad del diseño y el límite de 16 bloques. Los bloques persistidos se limitan a los módulos registrados por la aplicación. `window.BufaloInvoiceStudio.attachModule(id, descriptor)` sigue disponible para añadir módulos de vista en la sesión; guardar nuevos módulos requiere registrarlos y permitirlos en el servidor.

Las pruebas de servicios y controladores ejecutan la migración en PostgreSQL aislado y cubren la propiedad entre brokers, la elección del diseño, el flujo de creación y la vista del chofer. La prueba de interfaz del diseñador simula el API para validar la carga y guardado remoto sin acceder a cuentas reales.
