# Crear ÃƒÂ¡reas y usuarios

Inicia el backend:

```powershell
go run ./cmd/api
```

URL base: `http://localhost:8080`.
Swagger: `http://localhost:8080/swagger/`.
Estos endpoints no requieren token. EnvÃƒÂ­a `Content-Type: application/json`.

## 1. Crear un ÃƒÂ¡rea o departamento

```http
POST /api/v1/departments
```

Body:

```json
{
  "name": "Sistemas",
  "code": "SIS"
}
```

| Campo | Obligatorio | DescripciÃƒÂ³n |
| --- | --- | --- |
| name | SÃƒÂ­ | Nombre del ÃƒÂ¡rea, mÃƒÂ¡ximo 255 bytes. |
| code | SÃƒÂ­ | CÃƒÂ³digo ÃƒÂºnico, mÃƒÂ¡ximo 50 bytes. Se guarda en mayÃƒÂºsculas. |
| parent_id | No | IDs numÃ©ricos autogenerados de un ÃƒÂ¡rea superior existente y activa. Puede omitirse o ser null. |

Respuesta de ejemplo Ã¢â‚¬â€ `201 Created`:

```json
{
  "id": 7,
  "created_at": "2026-10-28T20:00:00Z",
  "updated_at": "2026-10-28T20:00:00Z",
  "name": "Sistemas",
  "code": "SIS",
  "parent_id": null,
  "is_active": true
}
```

Guarda el `id` devuelto: se utiliza como `department_id` al crear usuarios.

## 2. Crear un usuario y asignarlo al ÃƒÂ¡rea

```http
POST /api/v1/users
```

Body:

```json
{
  "name": "MarÃƒÂ­a",
  "email": "maria@example.com",
  "username": "maria",
  "password": "Example123!",
  "department_id": 7
}
```

| Campo | Obligatorio | DescripciÃƒÂ³n |
| --- | --- | --- |
| name | SÃƒÂ­ | Nombre del usuario, mÃƒÂ¡ximo 255 bytes. |
| email | SÃƒÂ­ | Correo vÃƒÂ¡lido y ÃƒÂºnico, mÃƒÂ¡ximo 255 bytes. Se guarda en minÃƒÂºsculas. |
| username | SÃƒÂ­ | ÃƒÅ¡nico, mÃƒÂ¡ximo 100 bytes. Admite letras, nÃƒÂºmeros, punto, guion y guion bajo. Se guarda en minÃƒÂºsculas. |
| password | SÃƒÂ­ | Entre 8 y 72 bytes. Se guarda como hash bcrypt en una columna binaria BYTEA. |
| department_id | SÃƒÂ­ | IDs numÃ©ricos autogenerados de un ÃƒÂ¡rea existente y activa. |

Respuesta de ejemplo Ã¢â‚¬â€ `201 Created`:

```json
{
  "user": {
    "id": "660e8400-e29b-41d4-a716-446655440001",
    "created_at": "2026-10-28T20:01:00Z",
    "updated_at": "2026-10-28T20:01:00Z",
    "username": "maria",
    "admin": false,
    "name": "MarÃƒÂ­a",
    "email": "maria@example.com",
    "is_active": true
  },
  "department_id": 7
}
```

El usuario y su pertenencia al ÃƒÂ¡rea se crean en una sola transacciÃƒÂ³n. El rol inicial en el ÃƒÂ¡rea es `MEMBER`. La contraseÃƒÂ±a y su hash no aparecen en la respuesta.

`admin` se crea en false. Este endpoint no permite asignarlo desde el body.

## Errores

| CÃƒÂ³digo HTTP | Motivo |
| --- | --- |
| 400 | JSON o campos invÃƒÂ¡lidos; ÃƒÂ¡rea inexistente o inactiva. |
| 409 | Username, email o cÃƒÂ³digo del ÃƒÂ¡rea duplicado. |
| 500 | Error interno al completar la operaciÃƒÂ³n. |

Formato:

```json
{
  "error": "DescripciÃƒÂ³n del error"
}
```

## Ejemplo para el frontend

```javascript
async function postJSON(path, body) {
  const response = await fetch(path, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body)
  });
  const result = await response.json();
  if (!response.ok) throw new Error(result.error || "No se pudo crear");
  return result;
}

const area = await postJSON("/api/v1/departments", {
  name: "Sistemas",
  code: "SIS"
});

const result = await postJSON("/api/v1/users", {
  name: "MarÃƒÂ­a",
  email: "maria@example.com",
  username: "maria",
  password: "Example123!",
  department_id: area.id
});

console.log(result.user);
```

Las rutas relativas del ejemplo requieren servir el frontend en el mismo origen o configurar un proxy de `/api` hacia el backend.

Para un formulario de usuario, llena el selector de ÃƒÂ¡reas con `GET /api/v1/departments?page=1&page_size=100&is_active=true`. Usa `id` como valor y `name` como etiqueta. Si hay mÃƒÂ¡s pÃƒÂ¡ginas, agrega bÃƒÂºsqueda o carga de pÃƒÂ¡ginas adicionales.




