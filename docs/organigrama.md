# ÃƒÂreas: padres y organigrama

## Listado paginado

`GET /api/v1/departments?page=1&page_size=20`

Cada registro agrega `parent` con `id`, `name`, `code` e `is_active`. Para ÃƒÂ¡reas superiores, `parent_id` y `parent` son null. El padre inmediato se incluye aunque estÃƒÂ© fuera de la pÃƒÂ¡gina o del filtro. Se conservan los filtros q e is_active. Los padres se consultan por lote.

## Organigrama general

`GET /api/v1/departments/organization-chart`

Respuesta esquemÃƒÂ¡tica (cada nodo tambiÃƒÂ©n incluye los campos normales del departamento):

```json
{
  "roots": [
    {
      "id": 1,
      "name": "DirecciÃƒÂ³n",
      "parent_id": null,
      "children": [
        {
          "id": 2,
          "name": "Sistemas",
          "parent_id": 1,
          "children": []
        }
      ]
    }
  ],
  "total": 2,
  "root_count": 1
}
```

roots son las ÃƒÂ¡reas superiores. Si root_count es 1 hay una ÃƒÂºnica ÃƒÂ¡rea superior; si es mayor, hay varias jerarquÃƒÂ­as independientes. Sin ÃƒÂ¡reas devuelve roots: [], total: 0 y root_count: 0.

El organigrama es completo y no paginado. Incluye activas e inactivas, identificadas con is_active. No aplica filtros. Ordena raÃƒÂ­ces y hermanos por nombre e ID. Devuelve 409 si encuentra ciclos o un padre ausente.

En el frontend, dibuja roots y luego los children de cada nodo recursivamente. No deduzcas la raÃƒÂ­z a partir del primer registro del listado paginado. Puedes mostrar ÃƒÂ¡reas inactivas con un estilo distinto conservando su posiciÃƒÂ³n.

Ambos GET estÃƒÂ¡n disponibles sin token por ahora. No se requieren migraciones nuevas.




