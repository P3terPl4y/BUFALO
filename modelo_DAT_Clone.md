El modelo de datos del Broker en DAT One no está publicado como un esquema técnico oficial (ERD o diccionario de datos), pero a partir de la documentación funcional y los flujos de la plataforma se puede reconstruir su estructura lógica completa. A continuación presento una síntesis de las entidades, atributos y relaciones que componen el modelo de datos del bróker dentro de DAT One.
🏢 Entidad: Broker

El bróker es el actor central que publica cargas, gestiona su red de transportistas y negocia tarifas.
Atributo	Descripción	Fuente
Broker ID	Identificador único interno del bróker en DAT One.	
Nombre de la empresa	Nombre legal o comercial del bróker.	
DOT Number	Número de registro del Departamento de Transporte de EE. UU.	
MC Number	Número de autoridad de operación como bróker (Property Broker).	
Estado de autoridad	Activo, Inactivo o Ninguno para la autoridad de bróker.	
Información de contacto	Teléfono, correo, dirección física.	
Credit Score	Puntuación de crédito del bróker visible para transportistas en planes avanzados.	
Days-to-Pay	Promedio de días que tarda el bróker en pagar a los transportistas.	
Preferido / Bloqueado	Etiquetas que el bróker asigna a transportistas en su red.	
Audiencia de publicación	Define si la carga se publica en todo el Load Board, en la red extendida o solo en la red privada.	
📦 Entidad: Load (Carga)

Representa cada envío publicado por el bróker.
Atributo	Descripción	Fuente
Load ID / Reference ID	Identificador que el bróker usa para rastrear la carga.	
Origen	Ciudad y estado de recogida. Incluye deadhead mileage (millas en vacío hasta el punto de recogida).	
Destino	Ciudad y estado de entrega.	
Fecha de recogida	Fecha en que debe recogerse la carga.	
Tipo de carga	Full truckload (FTL) o Partial truckload (LTL).	
Tipo de equipo	Dry van, flatbed, reefer, step deck, double drop, lowboy, etc.	
Longitud del remolque	Longitud requerida en pies.	
Peso	Peso de la carga en libras.	
Commodity	Tipo de mercancía (puede dejarse en blanco).	
Atributos de manejo especial	Para flatbed, step deck, etc. (ej. sobreancho, sobresaliente).	
Tarifa total	Monto total que el bróker está dispuesto a pagar.	
Tarifa por milla	Tarifa total dividida por las millas del viaje.	
Market Rate	Tarifa de mercado (disponible según suscripción).	
Spot Rate	Tarifa spot de referencia.	
Contract Rate	Tarifa contractual de referencia.	
Estado de publicación	Publicada, en negociación, cubierta, cancelada.	
🚛 Entidad: Carrier (Transportista)

Los transportistas son los receptores de las cargas y parte de la red del bróker.
Atributo	Descripción	Fuente
Carrier DOT Number	Identificador único del transportista.	
Nombre de la empresa	Nombre legal o comercial.	
MC Number	Número de autoridad de operación.	
Safety Rating	Calificación de seguridad de FMCSA (Satisfactory, Conditional, Unsatisfactory).	
Autoridad activa	Common, Contract o Broker activo.	
Información de seguro	COI (Certificate of Insurance) con límites de cobertura, VINs, proveedor.	
Tipo de equipo	Equipos que el transportista puede operar.	
Ubicación de oficina	Ciudad y estado.	
Actividad en la lane	Número de búsquedas de carga y publicaciones de camiones en un período.	
Etiqueta de red	In Network, Blocked, o sin etiqueta.	
Estado de ELD	Información de ELD, tracking y entregas a tiempo (en Company Profile).	
🧩 Entidad: Company Profile (Perfil de Compañía)

Es la vista consolidada de la información de un transportista o bróker dentro de DAT One.
Sección	Atributos	Fuente
Company Details	Doing-Business-As Name, Company Type, Legal Name.	
Safety Rating Details	Safety Rating de FMCSA.	
Authority Status	Authority Type (Common, Contract, Broker), Status (Active, Inactive, None), Application (pendientes).	
Authority Details	Tipo de operación (interstate/intrastate, hazardous/non-hazardous).	
Contact Info	Teléfono, correo, dirección.	
Insurance Info	COI, límites de cobertura, VINs, proveedor.	
Network Tags	In Network, Blocked.	
🔗 Relaciones entre Entidades
Relación	Cardinalidad	Descripción
Broker → Load	1 : N	Un bróker publica muchas cargas.
Broker → Carrier (Network)	N : M	Un bróker gestiona una red de transportistas (In Network, Blocked).
Load → Carrier	N : 1	Una carga es cubierta por un transportista (o queda vacante).
Carrier → Company Profile	1 : 1	Cada transportista tiene un perfil de compañía con información de FMCSA, seguro, etc.
Broker → Company Profile	1 : 1	El bróker también tiene un perfil con su información de autoridad y crédito.
Load → Rate	1 : 1	Cada carga tiene una tarifa asociada (total, por milla, market/spot/contract).
Carrier → Lane Activity	1 : N	Un transportista tiene múltiples registros de actividad en lanes (búsquedas, posteos de camiones).
📌 Consideraciones sobre el modelo

    No es un esquema público: DAT no publica un diagrama entidad-relación oficial. El modelo aquí presentado se infiere de la documentación de ayuda, los flujos de la interfaz y las descripciones de productos como LaneMakers, Network Management y Company Profile.

    Datos sensibles: Algunos atributos (Credit Score, Days-to-Pay, COI detallado) solo están disponibles en niveles de suscripción avanzados o mediante API específicas.

    Integración vía API: DAT ofrece endpoints REST para postear cargas, consultar tarifas y buscar capacidad, lo que implica que el modelo subyacente está expuesto de forma controlada a través de esas APIs.

    Evolución: El modelo se amplía continuamente con nuevas categorías de equipos (cargo van, box truck, power-only) y atributos de manejo especial.

Si necesitas profundizar en alguna entidad concreta o en los campos exactos de la API, puedo ayudarte a extraer más detalles de la documentación disponible.
