# Refinamiento de empresas y acceso al chat

## Objetivo

Reducir pasos entre las empresas a las que una persona ya pertenece y el chat compartido, hacer visible la identidad de cada empresa y evitar solicitudes de afiliación inválidas.

## Decisiones de UX

1. **Inicio en “Mis empresas”.** El listado inicial contiene empresas que el usuario creó o a las que está asociado, incluso si el vínculo histórico solo aparece en el perfil de chofer/publicador.
2. **Dos destinos claros por tarjeta.** El nombre y la acción principal abren el chat cuando la cuenta pertenece a la empresa. El logo siempre abre la ficha informativa de la empresa. Los enlaces tienen destinos separados y accesibles por teclado.
3. **Búsqueda bajo demanda.** “Buscar empresas” abre un estado vacío con el filtro enfocado visualmente. No se descargan ni presentan directorios globales hasta que el usuario consulta un nombre, MC o DOT. La búsqueda solo incluye empresas activas del tipo compatible con su rol y excluye las que ya tiene asociadas.
4. **Estados explícitos de afiliación.** Los controles de solicitar, pendiente, miembro y propietario no se muestran juntos ni se contradicen. Los propietarios no reciben acciones de afiliación; la regla también se impone en el servicio.
5. **Identidad de empresa consistente.** Cada empresa tiene su propia imagen de perfil, que se muestra en sus tarjetas y ficha. Se puede cargar al crear o editar; el formulario acepta JPEG/PNG de hasta 5 MB y 16 MP.
6. **Edición con propiedad clara.** El creador (`owner_id`) administra los datos y la imagen de la empresa. Se conserva el acceso del administrador global para no romper las operaciones administrativas existentes.

## Refactorización y validación

- Mantener las tarjetas, tipografía, espaciado, estados y colores de BUFALO; añadir solo estilos acotados para la imagen y los accesos de empresa.
- Separar en el controlador la vista de empresas asociadas de la búsqueda del directorio; no cambiar los perfiles ni las solicitudes ya existentes.
- Aplicar autorización en servicio y controlador. La interfaz mejora el descubrimiento, pero nunca sirve como control de permisos.
- Añadir `profile_photo` como columna nullable para que datos anteriores sigan funcionando con el icono de empresa predeterminado.
- Guardar imágenes con nombre aleatorio, validación real de contenido y dimensiones, límite de lectura y borrado seguro únicamente del archivo anterior bajo el directorio de logos.
- Cubrir con pruebas: propietario, miembro, solicitante elegible, solicitud pendiente, empresa ya afiliada, búsqueda vacía y filtrada, enlace a chat, enlace a ficha, y rechazo de afiliación de propietarios.
- Verificar carga adaptable, teclado, enlaces con destinos claros, formulario multipart, CSRF, permisos de edición y que la edición no pueda cambiar `owner_id`.

## Mejoras siguientes recomendadas

- Añadir un estado de solicitud “rechazada” en la ficha personal con acción “Volver a solicitar” cuando la política lo permita; hoy el usuario puede iniciar una nueva solicitud, pero el historial no está presentado como línea de tiempo.
- Mostrar un contador discreto de mensajes no leídos en la tarjeta de cada empresa cuando BUFALO tenga un estado de lectura de chat por usuario.
- Permitir recortar una imagen cuadrada en el navegador antes de cargarla, manteniendo validación del archivo en backend.
- Si una cuenta puede representar más de una empresa, ofrecer un selector de espacio de trabajo con empresa activa persistente; para el modelo actual, la lista de tarjetas es más simple y evita introducir estado de sesión adicional.

## Implementación de esta iteración

Esta iteración implementa las decisiones 1–6, la migración nullable, el bloqueo de afiliación a nivel de servicio y pruebas de acceso y renderizado. La revisión visual autenticada en producción debe completarse con el navegador del usuario después del despliegue.
