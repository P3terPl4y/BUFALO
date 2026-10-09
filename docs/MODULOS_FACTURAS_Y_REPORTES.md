# Módulos de informes y diseño de facturas

## Salud e informes

La vista `/admin/health` y las exportaciones parten de `healthSnapshot`. Esto mantiene iguales la lectura de servicios, contadores, cola de notificaciones, protección de tráfico, tendencias e historial. El exportador recibe secciones con encabezados y filas, y las presenta en Excel o CSV respetando el tab seleccionado y los filtros de estado, método y ruta.

PDF abre la impresión del navegador sobre la vista de salud existente. Así conserva la maquetación visible, sus tarjetas, gráficas y tabla; la hoja de impresión oculta la navegación y los controles. El usuario puede guardar esa vista como PDF desde el diálogo de impresión.

`app/exports.Section` es la unidad extensible para agrupar un informe. Los módulos pueden añadir secciones sin alterar los generadores CSV/XLSX/PDF. XLSX comparte los encabezados BUFALO y conserva la primera fila al desplazarse. CSV lleva BOM UTF-8 y neutraliza texto que Excel interpretaría como fórmula.

## Laboratorio de factura

`/facturas/disenador` y `/admin/facturas/disenador` comparten una única vista. El módulo no crea ni emite facturas reales. Usa datos de demostración y persiste hasta doce diseños locales en el navegador, nunca información de facturas.

Cada bloque aporta sus metadatos y su función de renderizado. `window.BufaloInvoiceStudio.attachModule(id, descriptor)` acopla un bloque nuevo; `detachModule(id)` lo retira del catálogo y de los diseños abiertos. Se permiten hasta dieciséis módulos y dieciséis bloques por diseño. Los diseños conservan los identificadores válidos de módulos que todavía no se hayan cargado y los muestran cuando el módulo se acopla; esto permite cargar extensiones después del editor sin perder configuraciones guardadas. El lienzo, orden, presets, persistencia e impresión no necesitan conocer el contenido de los bloques. Los renderizadores son código confiable de la aplicación; deben escapar cualquier valor de usuario y validar listas antes de incorporarlo a la vista.

Los diseños guardados son deliberadamente locales a cada navegador: este entorno de prueba no añade migraciones ni modifica el flujo de emisión. La conexión posterior a plantillas compartidas debe añadir almacenamiento autenticado y versionar el esquema antes de usar los diseños para emitir documentos.

La prueba de interfaz aislada del diseñador se ejecuta con `cd tests/browser && npm install && npm run test:invoice`. Usa un documento de demostración y almacenamiento simulado; no requiere ni toca una cuenta ni una base de datos.
