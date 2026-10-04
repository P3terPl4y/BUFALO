# BUFALO Móvil

Aplicación web progresiva (PWA) hecha con React. Se sirve desde el mismo
origen que BUFALO para reutilizar la sesión, las reglas de acceso y la
protección CSRF existentes.

## Desarrollo y compilación

Ejecuta npm install y npm run build desde esta carpeta. Vite genera los
archivos servidos por Fiber en ../public/mobile. Abre /mobile/ en el navegador.
La app necesita que el backend BUFALO esté activo.

Para generar una APK de depuración en un entorno Android preparado, ejecuta
npm run android:debug. Se guarda en
android/app/build/outputs/apk/debug/app-debug.apk. El APK abre una pantalla
para introducir el dominio o la IP LAN del servidor BUFALO; en un teléfono no
debe usarse localhost para apuntar al ordenador.

## Experiencia incluida

- Inicio de sesión compatible con la autenticación web actual.
- Resumen y búsqueda de cargas por rol, con rutas, fechas, peso y tarifa.
- Para choferes: aceptar cargas disponibles, iniciar el tránsito y registrar
  entregas. El servidor vuelve a validar propiedad, rol y estado en cada paso.
- Accesos a facturas, direcciones, administración y edición del perfil en los
  módulos web existentes.
- Diseño adaptable desde 320 px, navegación táctil, áreas seguras de iOS,
  modo instalable y caché del shell estático. Las operaciones requieren red.

## Instalación

En Android/Chrome usa «Instalar app» cuando aparezca. En iOS/Safari elige
Compartir → Añadir a pantalla de inicio. La instalación requiere HTTPS en el
entorno desplegado; localhost se admite durante desarrollo.
