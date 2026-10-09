# Plan de refinamiento UX/UI de las nuevas funciones

**Alcance:** auditoría heurística de notificaciones, afiliaciones, chat de empresa, tarjetas de entidades y estudio de facturas, contrastada con el sistema visual actual de BUFALO. La primera entrega implementa el centro de notificaciones y añade atribución de autor a los eventos nuevos. La auditoría de código no sustituye pruebas de usabilidad con choferes y publicadores reales.

## Criterio de diseño

BUFALO ya define una identidad útil: fondos claros, texto grafito, ámbar para acciones y estados de atención, superficies blancas, radios moderados e iconos Bootstrap. Las nuevas pantallas deben reutilizar esos tokens (`--tf-*`), el encabezado de página (`tf-page-header`) y las acciones existentes. No deben introducir una segunda paleta, gradientes decorativos generalizados ni animaciones que compitan con información operativa.

La jerarquía sigue este orden: qué ocurrió, quién intervino, cuándo ocurrió y qué puede hacer el usuario. Cada estado debe tener texto e icono además de color, los enlaces deben llevar al objeto nombrado y las acciones deben conservar etiquetas explícitas y objetivos cómodos para teclado y móvil.

## Auditoría heurística

| Superficie | Fricción observada | Mejora prioritaria |
| --- | --- | --- |
| Bandeja de notificaciones | Tarjetas genéricas; no identifican a la persona que originó el evento, no distinguen bien lo leído y lo pendiente y ofrecen solo un enlace textual a la carga. | Autor con avatar y perfil cuando exista; tipo y fecha legibles; estado no leído persistente; enlace directo al recurso; resumen de pendientes y acción para marcar leído. Mantener el estado vacío sobrio. |
| Afiliación de empresa | Solicitudes presentadas como una línea con correo y botones juntos; no hay jerarquía entre identidad, fecha y decisión. | Tarjeta de solicitud con nombre/foto/rol, fecha y acciones con etiquetas completas. La aprobación es la acción principal; el rechazo queda claramente separado. El perfil se abre en una ruta existente con sus controles de privacidad. |
| Chat de empresa | Autor y hora comparten una línea de texto pequeña; el autor no tiene avatar ni perfil enlazado. La interfaz dice “se actualiza automáticamente” aunque el cliente consulta cada 30 segundos. | Identidad visual constante, perfil del autor si está disponible, mensajes propios y ajenos diferenciados con estructura (no solo color), estado de moderación explícito y texto honesto sobre la actualización periódica. Conservar el historial y el envío actuales. |
| Estudio de facturas | Paleta, vista previa y propiedades aparecen simultáneamente en tres columnas; para una tarea de diseño puede ser difícil saber qué hacer primero. | Guiar el flujo en pasos pequeños: escoger formato, organizar bloques, revisar contenido y guardar. Mantener una vista previa persistente; agrupar propiedades avanzadas y ofrecer controles de teclado equivalentes al arrastre. |
| Tarjetas y tablas de personas/empresas | El selector cards/tabla está implementado y ayuda a elegir densidad; falta auditar consistencia en acciones, nombres largos, estados vacíos y lectura en móvil entre todos los apartados. | Consolidar un componente de identidad reutilizable, fijar un orden de acciones común y preservar la preferencia de vista por sección. No migrar todo de una vez: inventariar y validar pantalla a pantalla. |

## Secuencia de refactorización

1. **Inventario y límites:** registrar rutas, roles, datos existentes, acciones, permisos y estados vacíos/cargando/error. No cambiar servicios de negocio salvo que falte contexto necesario para presentar un evento correctamente.
2. **Contrato de información:** definir en cada vista qué entidad ocurrió, actor, fecha, estado, destino y acción permitida. Para notificaciones, guardar el actor en el mismo commit que el evento y mantener `NULL` para registros históricos o eventos del sistema.
3. **Estructura y jerarquía:** ordenar la información de más urgente a complementaria, reutilizar componentes/tokens existentes y mantener una única acción primaria por tarjeta.
4. **Responsive y accesibilidad:** verificar anchos de 320–1440 px, teclado, foco visible, etiquetas de controles, contraste y estados no comunicados solo mediante color. Reducir movimiento cuando el usuario lo solicite.
5. **Validación funcional:** pruebas de plantilla con cada tipo de evento, actor con y sin perfil, recursos de carga/empresa, leído/no leído y estado vacío. Probar además permisos del destino: la ruta debe seguir aplicando las reglas que ya protegen datos.
6. **Revisión visual y rollout:** captura antes/después con datos ficticios, comparación en escritorio y móvil, ejecución de pruebas funcionales y de seguridad existentes y despliegue reversible por una superficie cada vez. Revisar métricas de errores y comentarios de choferes/publicadores antes de extender el patrón.

## Entrega implementada en esta iteración

- La bandeja destaca tipo de evento, hora, autor y estado de lectura; muestra avatar si existe, enlaza al perfil profesional correspondiente y ofrece acceso directo a carga o empresa cuando el registro tiene ese recurso.
- Se incorpora `actor_user_id` nullable para eventos nuevos. Los registros existentes permanecen válidos y aparecen atribuidos a BUFALO cuando no hay autor guardado.
- Se expone el conteo de no leídas y se añade acción de marcar como leída con CSRF. El modelo visual mantiene la paleta ámbar/grafito y no añade ilustración al estado vacío.
- Afiliaciones, chat, estudio de factura y tablas/cards quedan priorizados arriba para continuar con la misma secuencia, sin una reescritura masiva.

## Referencias consultadas

- [Nielsen Norman Group: indicadores, validaciones y notificaciones](https://www.nngroup.com/articles/indicators-validations-notifications/): elegir el canal según urgencia y mantener visible el estado del sistema.
- [Nielsen Norman Group: notificaciones útiles y específicas](https://www.nngroup.com/articles/smart-home-notifications/): mensajes relevantes, oportunos y concretos reducen fatiga y pérdida de confianza.
- [Nielsen Norman Group: notificaciones autosuficientes](https://www.nngroup.com/articles/mobile-microsessions/): incluir suficiente contexto para entender el aviso y decidir si abrirlo.
- [GOV.UK Design System: notification banner](https://design-system.service.gov.uk/components/notification-banner/): alinear mensajes con el contenido, evitar avisos redundantes y usar roles semánticos accesibles.
