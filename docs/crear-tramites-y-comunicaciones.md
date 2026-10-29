# Tramites y comunicaciones

Base: `/api/v1`. Todos los endpoints son publicos; no usan JWT, token ni header `Authorization`. Los IDs son numeros JSON.

Identificacion del usuario que hace la peticion:

- Crear tramite, crear mensaje, responder y reenviar: `created_by` en el body.
- Todo lo demas (GET, cerrar tramite, marcar leido, cambiar estado): query param `?user_id={id}`. Ejemplo: `GET /messages/inbox/personal?user_id=12`.

Con ese usuario el backend valida permisos (si pertenece al area, si es destinatario, etc.).

## POST /procedures

Tramite personal:

```json
{ "name": "Revision", "description": "Revision del expediente del caso 2026-015 antes de su aprobacion.", "procedure_type": "PERSONAL", "created_by": 12 }
```

Tramite de departamento:

```json
{ "name": "Compra", "description": "Solicitud de compra de 10 laptops para el area de Sistemas.", "procedure_type": "DEPARTMENT", "sender_department_id": 7, "created_by": 12 }
```

Campos:

- `name`: titulo corto del tramite.
- `description`: breve explicacion de que trata el tramite y su objetivo, para que los destinatarios entiendan el contexto. Cumple el mismo papel que `body` en los mensajes.

El backend registra siempre `created_by`. En PERSONAL tambien registra `sender_user_id = created_by`; en DEPARTMENT registra `sender_department_id`.

Response 201:

```json
{ "id": 42, "procedure_number": "TRAMITE/SYS/3", "name": "Compra", "description": "Solicitud de compra de 10 laptops para el area de Sistemas.", "procedure_type": "DEPARTMENT", "created_by": 12, "sender_user_id": null, "sender_department_id": 7, "status": "OPEN" }
```

### Numero de tramite (`procedure_number`)

Cada tramite tiene dos identificadores:

- `id`: ID numerico interno, igual que en los mensajes.
- `procedure_number`: codigo legible que se muestra al usuario.

El `procedure_number` lo genera el backend automaticamente al crear el tramite. **El frontend no lo envia** en el `POST`; solo lo recibe en la respuesta y en todos los `GET`.

| Tipo | Formato | Ejemplo | Partes |
| --- | --- | --- | --- |
| PERSONAL | `PERSONAL/{correlativo}/{codigo_usuario}` | `PERSONAL/01/0101` | `01`: correlativo de tramites de esa persona. `0101`: codigo de la persona (su ID). |
| DEPARTMENT | `TRAMITE/{siglas_area}/{correlativo}` | `TRAMITE/SYS/1`, `TRAMITE/SYS/2` | `SYS`: abreviatura del area. `1, 2, 3...`: correlativo de tramites de esa area. |

Reglas:

- El correlativo es independiente por persona (PERSONAL) y por area (DEPARTMENT): Sistemas tiene sus propios 1, 2, 3 y Contabilidad tiene los suyos.
- El correlativo aumenta solo y nunca se repite ni se renumera.
- Es de solo lectura: no se puede editar desde el frontend.

Uso en el frontend: mostrar `procedure_number` en listados, detalle y selectores de tramite (por ejemplo, al asociar un mensaje a un tramite), y seguir usando `id` para las llamadas a la API (`procedure_id`).

## POST /messages

`status` admite `BORRADOR` y `HECHO`. Si se omite, el API usa `HECHO` para
mantener compatibilidad con los clientes actuales. Un `BORRADOR` queda con
`sent_at: null` y puede no incluir asunto, cuerpo ni destinatarios; un `HECHO`
requiere asunto, cuerpo y al menos un destinatario `TO` y registra `sent_at`.

Las respuestas se vinculan exclusivamente mediante `message_relations`; usa
`POST /messages/{id}/reply` para crearlas.

Mensaje personal:

```json
{ "created_by": 12, "message_type": "PERSONAL", "visibility": "PRIVATE", "subject": "Consulta", "body": "Revisar caso.", "recipients": [{ "recipient_id": 15, "type": "TO" }], "document_ids": [] }
```

Mensaje de departamento:

```json
{ "created_by": 12, "message_type": "DEPARTMENT", "sender_department_id": 7, "procedure_id": 42, "visibility": "DEPARTMENT", "subject": "Revision", "body": "Revisar adjunto.", "recipients": [{ "department_id": 9, "type": "TO" }], "document_ids": [81] }
```

Response 201:

```json
{ "status": "created" }
```

### Flujo: mensajes dentro de un tramite

1. Se crea el tramite (`POST /procedures`).
2. El usuario entra al tramite (`GET /procedures/{id}?user_id=12`).
3. Para comunicar algo, crea un mensaje con `procedure_id` del tramite y sus `recipients` (`POST /messages`).
4. Los mensajes del tramite se listan con `GET /procedures/{id}/messages?user_id=12`.

### Destinatarios (`recipients` / tabla `message_recipients`)

Cada mensaje guarda en `message_recipients` una fila por destinatario. **Un destinatario puede ser una persona o un area completa**, y se pueden mezclar en el mismo mensaje.

| Campo | Descripcion |
| --- | --- |
| `id` | ID del registro destinatario (se usa en `/message-recipients/{id}/...`). |
| `message_id` | Mensaje al que pertenece. |
| `recipient_id` | ID del usuario destinatario. `null` si el destinatario es un area. |
| `department_id` | ID del area destinataria. `null` si el destinatario es una persona. |
| `type` | `TO` (destinatario principal) o `CC` (con copia). |
| `status` | `PENDING`, `READ`, `IN_PROGRESS`, `ANSWERED`, `CLOSED`. Inicia en `PENDING`. |
| `read_at` | Fecha de lectura (solo destinatarios persona). |

Reglas:

- Cada destinatario lleva **exactamente uno** de `recipient_id` o `department_id`, nunca ambos ni ninguno.
- Debe haber al menos un destinatario `TO`.
- No se puede repetir la misma persona ni la misma area en un mensaje.

Ejemplo con una persona, un area y una copia a otra area:

```json
{
  "created_by": 12,
  "message_type": "DEPARTMENT",
  "sender_department_id": 7,
  "procedure_id": 42,
  "visibility": "DEPARTMENT",
  "subject": "Revision de cotizacion",
  "body": "Adjunto la cotizacion para su revision.",
  "recipients": [
    { "recipient_id": 15, "type": "TO" },
    { "department_id": 9, "type": "TO" },
    { "department_id": 4, "type": "CC" }
  ],
  "document_ids": [81]
}
```

Cuando el destinatario es un area:

- Todos los miembros activos del area ven el mensaje en `GET /messages/inbox/department?user_id={id}`.
- La lectura se registra por cada usuario en `message_recipient_reads`, es decir, quien del area lo leyo y cuando. El `status` del destinatario es compartido por el area.

Cuando el destinatario es una persona, el mensaje aparece en `GET /messages/inbox/personal?user_id={id}`.

Acciones sobre un destinatario:

- `POST /message-recipients/{id}/read?user_id=15`: marca como leido (`204`).
- `PATCH /message-recipients/{id}/status?user_id=15` con `{ "status": "IN_PROGRESS" }`: cambia el estado (`200`).

Solo puede hacerlo la persona destinataria o un miembro del area destinataria; si no, `403`.

## Listados por remitente

Paginacion: `page` y `page_size` (1-100).

- `GET /procedures/by-sender-department/{departmentId}?page=1&page_size=20&status=OPEN&q=compra&user_id=12`
- `GET /procedures/by-sender-user/{userId}?page=1&page_size=20&status=OPEN&q=revision&user_id=12`
- `GET /messages/by-sender-department/{departmentId}?page=1&page_size=20&message_type=DEPARTMENT&visibility=DEPARTMENT&procedure_id=42&q=revision&user_id=12`
- `GET /messages/by-sender-user/{userId}?page=1&page_size=20&message_type=PERSONAL&q=consulta&user_id=12`

Los cuatro devuelven `200`:

```json
{ "data": [], "total": 0, "page": 1, "page_size": 20 }
```

En los listados de tramites, cada elemento de `data` incluye `procedure_number`:

```json
{ "data": [{ "id": 42, "procedure_number": "TRAMITE/SYS/3", "name": "Compra", "description": "Solicitud de compra de 10 laptops para el area de Sistemas.", "procedure_type": "DEPARTMENT", "status": "OPEN" }], "total": 1, "page": 1, "page_size": 20 }
```

El filtro `q` tambien busca por `procedure_number` (por ejemplo `q=TRAMITE/SYS`).

Errores: `400` para campos/IDs invalidos o tramite cerrado; `403` al intentar representar un departamento sin el rol requerido.

