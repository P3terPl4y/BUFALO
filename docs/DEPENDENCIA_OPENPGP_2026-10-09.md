# Aviso GO-2026-5932

El análisis de módulos identifica `golang.org/x/crypto@v0.57.0` porque incluye el paquete `openpgp`. El aviso oficial abarca todas las versiones de ese paquete y no ofrece versión corregida: https://pkg.go.dev/vuln/GO-2026-5932.

BUFALO utiliza bcrypt de ese módulo para conservar compatibilidad con contraseñas existentes. Retirar todo el módulo rompería el login; sustituir contraseñas por un algoritmo casero no sería una corrección.

La mitigación consiste en comprobar el grafo de paquetes realmente importados, mantener el análisis de símbolos y hacer fallar CI si aparece una importación de `golang.org/x/crypto/openpgp` o cualquiera de sus subpaquetes. No se suprime ni se oculta el aviso del inventario de módulos. Se conserva su identificación para las revisiones futuras. La ausencia de importación debe verificarse en cada candidato, no inferirse de un análisis antiguo.
