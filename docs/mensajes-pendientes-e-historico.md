# Mensajes pendientes e historico

Base: `/api/v1`. Actualmente la identidad se envia como `user_id` porque el
backend no usa JWT.

## 1. Ver mensajes pendientes de lectura

`GET /message-recipients?user_id={id}&page=1&page_size=20`

Devuelve solo los registros de destinatario que pertenecen al usuario, ya sea
directamente o por una de sus areas activas, y que aun no ha leido. Todos se
exponen con `status: "UNREAD"`. Los borradores no aparecen.

Ejemplo de respuesta:

```json
{
  "data": [{
    "id": 88,
    "message_id": 120,
    "recipient_id": 5,
    "department_id": null,
    "type": "TO",
    "status": "UNREAD",
    "subject": "Actualizacion del tramite",
    "created_at": "2026-10-29T10:00:00Z"
  }],
  "total": 1,
  "page": 1,
  "page_size": 20
}
```

Con el `message_id` se obtiene el contenido completo:

`GET /messages/120?user_id=5`

Para registrar la lectura despues de abrirlo:

`POST /message-recipients/88/read?user_id=5`

## 2. Ver el historico de un mensaje

`GET /messages/{messageId}/history?user_id={id}`

Devuelve la linea completa de mensajes conectados mediante `message_relations`,
incluyendo respuestas, reenvios y referencias. `messages` esta ordenado por
fecha y `relations` indica cada enlace con `source_message_id`,
`target_message_id` y `relation_type` (`REPLY`, `FORWARD` o `REFERENCE`). Solo
se devuelven mensajes y enlaces que el usuario puede consultar.

Ejemplo:

```json
{
  "message_id": 120,
  "messages": [{ "id": 120, "subject": "Solicitud" }, { "id": 135, "subject": "Respuesta" }],
  "relations": [{
    "source_message_id": 135,
    "target_message_id": 120,
    "relation_type": "REPLY"
  }]
}
```

`message_relations` es la unica fuente de verdad del hilo. No existe
`message_parent_id`, por lo que no se duplican enlaces ni se corre el riesgo de
que ambos datos queden inconsistentes.

## 3. Ver un tramite con todos sus mensajes

`GET /procedures/{procedureId}?user_id={id}`

Devuelve el tramite, sus mensajes visibles, todos los destinatarios y estados
de esos mensajes en `message_recipients`, y los enlaces entre ellos en
`message_relations`. Los mensajes se ordenan de forma ascendente por fecha para
reconstruir la secuencia del tramite.

```json
{
  "procedure": { "id": 42, "status": "OPEN" },
  "messages": [{ "id": 120, "status": "HECHO", "subject": "Solicitud" }],
  "message_recipients": [{ "message_id": 120, "status": "READ", "type": "TO" }],
  "message_relations": [{
    "source_message_id": 135,
    "target_message_id": 120,
    "relation_type": "REPLY"
  }]
}
```
