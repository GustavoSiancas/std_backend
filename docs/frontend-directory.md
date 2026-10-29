# Listados de administraciÃ³n

GET /api/v1/users?page=1&page_size=20&department_id=IDs numéricos autogenerados&q=maria

Filtros opcionales: department_id (pertenencia activa), q (nombre, username o email), admin e is_active (booleanos). El listado incluye activos e inactivos si no se especifica is_active.

GET /api/v1/departments?page=1&page_size=20&is_active=true&q=sis

Filtros opcionales: q (nombre o cÃ³digo), is_active. Ambos endpoints ordenan por nombre e ID y aceptan page desde 1 y page_size entre 1 y 100. La respuesta contiene data, page, page_size, total y total_pages. Una pÃ¡gina vacÃ­a devuelve data: [].

Cada usuario incluye departments: [{id, name, code, role, is_active, membership_active}]. Incluso al filtrar por un Ã¡rea, se muestran todas sus pertenencias. No se devuelven hashes ni contraseÃ±as.

## Consumo desde el frontend

Use un proxy del frontend hacia localhost:8080 para /api; asÃ­ las llamadas pueden ser relativas:

```js
async function getJSON(path, params, signal) {
  const response = await fetch(path + "?" + new URLSearchParams(params), {signal});
  const result = await response.json();
  if (!response.ok) throw new Error(result.error || "Error al consultar");
  return result;
}

async function loadUsers({page = 1, departmentId = "", search = ""}, signal) {
  const params = {page: String(page), page_size: "20"};
  if (departmentId) params.department_id = departmentId;
  if (search.trim()) params.q = search.trim();
  return getJSON("/api/v1/users", params, signal);
}

async function loadDepartmentOptions(page = 1, search = "", signal) {
  return getJSON("/api/v1/departments",
    {page: String(page), page_size: "50", is_active: "true", q: search}, signal);
}
```

Al abrir la pantalla cargue usuarios y la primera pÃ¡gina de Ã¡reas. El selector usa id como valor y name como etiqueta; agregue una opciÃ³n Todas las Ã¡reas que omita department_id. Para mÃ¡s de 50 Ã¡reas, use bÃºsqueda y cargar mÃ¡s con las pÃ¡ginas siguientes.

Al cambiar Ã¡rea o bÃºsqueda, vuelva a page=1. Dibuje las filas con data y muestre user.departments.map(d => d.name).join(", "). Use total_pages para habilitar anterior/siguiente; no calcule el total a partir de data.length. Aplique debounce a bÃºsqueda y cancele peticiones anteriores con AbortController para evitar resultados desactualizados. Muestre estados de carga, error y sin resultados.

Estos GET son pÃºblicos por ahora. user.admin permite elegir la pantalla, pero no constituye autorizaciÃ³n del servidor.


