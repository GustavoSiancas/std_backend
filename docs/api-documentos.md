# API de documentos

Base: http://localhost:8080/api/v1

Los siguientes endpoints no requieren token por ahora. Swagger: http://localhost:8080/swagger/.

## POST /documents ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Â Crear documento

Header: Content-Type: application/json

Request (sustituye los IDs numÃƒÂ©ricos autogenerados por IDs reales):

```json
{
  "document_type_id": 3,
  "department_id": 7,
  "created_by": 12,
  "name": "Solicitud de informaciÃƒÆ’Ã‚Â³n",
  "description": "DescripciÃƒÆ’Ã‚Â³n opcional",
  "file_url": "https://example.com/carta.pdf"
}
```

Solo description es opcional. Tipo, ÃƒÆ’Ã‚Â¡rea y usuario deben estar activos; el usuario debe pertenecer al ÃƒÆ’Ã‚Â¡rea. created_by se toma de user.id del login y department_id de su ÃƒÆ’Ã‚Â¡rea seleccionada. No requiere admin.

La URL debe ser HTTP/HTTPS absoluta. Se guarda el enlace; no se sube ni descarga el PDF.

Respuesta 201 (IDs ilustrativos):

```json
{
  "id": 42,
  "created_at": "2026-10-28T23:00:00-05:00",
  "updated_at": "2026-10-28T23:00:00-05:00",
  "document_type_id": 3,
  "document_number": "STD-1-SIS",
  "document_sequence_id": 18,
  "correlative_number": 1,
  "created_by": 12,
  "department_id": 7,
  "name": "Solicitud de informaciÃƒÆ’Ã‚Â³n",
  "description": "DescripciÃƒÆ’Ã‚Â³n opcional",
  "file_url": "https://example.com/carta.pdf"
}
```

document_number es el código visible STD-<consecutivo>-<código de departamento>; document_sequence_id es la referencia numérica interna al contador. correlative_number es el nÃƒÆ’Ã‚Âºmero reservado para ese documento. El backend genera ambos; no los envÃƒÆ’Ã‚Â­es. Si el tipo no lleva correlativo, ambos son null. La reserva y la creaciÃƒÆ’Ã‚Â³n son transaccionales.

## GET /documents ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Â Listar documentos

Ejemplo:

```http
GET /api/v1/documents?page=1&page_size=20&sort_by=created_at&order=desc
```

| ParÃƒÆ’Ã‚Â¡metro opcional | Uso |
| --- | --- |
| page | Desde 1; predeterminado 1. |
| page_size | Entre 1 y 100; predeterminado 20. |
| document_type_id | IDs numÃƒÂ©ricos autogenerados del tipo documental. |
| department_id | IDs numÃƒÂ©ricos autogenerados del ÃƒÆ’Ã‚Â¡rea. |
| created_by | IDs numÃƒÂ©ricos autogenerados del usuario creador. |
| year | AÃƒÆ’Ã‚Â±o del catÃƒÆ’Ã‚Â¡logo, entre 1 y 9999. Excluye documentos sin catÃƒÆ’Ã‚Â¡logo. |
| q | BÃƒÆ’Ã‚Âºsqueda parcial por nombre, descripciÃƒÆ’Ã‚Â³n, nombre o cÃƒÆ’Ã‚Â³digo del tipo. |
| sort_by | id, name, created_at, updated_at, correlative_number, document_type, department, year. Predeterminado created_at. |
| order | asc o desc; predeterminado desc. |

Filtros combinables. Orden estable con ID como desempate.

Respuesta 200:

```json
{
  "data": [
    {
      "id": 42,
      "created_at": "2026-10-28T23:00:00-05:00",
      "updated_at": "2026-10-28T23:00:00-05:00",
      "document_type_id": 3,
      "document_number": "STD-1-SIS",
      "document_sequence_id": 18,
  "correlative_number": 1,
      "created_by": 12,
      "department_id": 7,
      "name": "Solicitud de informaciÃƒÆ’Ã‚Â³n",
      "description": "DescripciÃƒÆ’Ã‚Â³n opcional",
      "file_url": "https://example.com/carta.pdf",
      "document_type": {
        "id": 3,
        "name": "Carta",
        "code": "CARTA",
        "has_correlative": true
      },
      "department": {
        "id": 7,
        "name": "Sistemas",
        "code": "SIS",
        "is_active": true
      },
      "year": 2026
    }
  ],
  "page": 1,
  "page_size": 20,
  "total": 1,
  "total_pages": 1
}
```

Sin resultados, data es []. Incluye documentos de todos los usuarios mientras no se configure autenticaciÃƒÆ’Ã‚Â³n. Para la vista personal envÃƒÆ’Ã‚Â­a created_by y/o department_id explÃƒÆ’Ã‚Â­citamente; son filtros, no permisos.

## GET /documents/:id ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Â Consultar documento

Devuelve 200 con un objeto igual al elemento de data del listado, con tipo, ÃƒÆ’Ã‚Â¡rea y aÃƒÆ’Ã‚Â±o. Devuelve 400 para IDs numÃƒÂ©ricos autogenerados invÃƒÆ’Ã‚Â¡lido y 404 si no existe.

## GET /document-types ÃƒÂ¢Ã¢â€šÂ¬Ã¢â‚¬Â Tipos de documentos

```http
GET /api/v1/document-types?page=1&page_size=20&is_active=true&sort_by=name&order=asc
```

Filtros: q (nombre o cÃƒÆ’Ã‚Â³digo), is_active y has_correlative (true/false). PaginaciÃƒÆ’Ã‚Â³n igual al listado de documentos. sort_by acepta id, name, code, created_at; predeterminado name. order acepta asc/desc; predeterminado asc.

Respuesta 200:

```json
{
  "data": [
    {
      "id": 3,
      "created_at": "2026-10-28T23:00:00-05:00",
      "updated_at": "2026-10-28T23:00:00-05:00",
      "name": "Carta",
      "code": "CARTA",
      "has_correlative": true,
      "is_active": true
    }
  ],
  "page": 1,
  "page_size": 20,
  "total": 1,
  "total_pages": 1
}
```

El catÃƒÆ’Ã‚Â¡logo inicial contiene Carta, Oficio, Informe, MemorÃƒÆ’Ã‚Â¡ndum, MemorÃƒÆ’Ã‚Â¡ndum mÃƒÆ’Ã‚Âºltiple y Carta mÃƒÆ’Ã‚Âºltiple.

## Frontend

Carga tipos activos para el selector y usa su id como document_type_id. Carga las ÃƒÆ’Ã‚Â¡reas desde user.departments del login o GET /departments. EnvÃƒÆ’Ã‚Â­a el POST con user.id como created_by.

Para la tabla muestra document_type.name, department.name, correlative_number y year. Abre file_url como enlace. Al cambiar filtros u orden reinicia page=1 y utiliza total_pages para anterior/siguiente. Refresca el listado al crear un documento. Los campos nuevos del GET se calculan con JOIN, sin una consulta adicional por documento.

## Errores

Todos usan {"error":"DescripciÃƒÆ’Ã‚Â³n"}. Los GET devuelven 400 ante filtros, paginaciÃƒÆ’Ã‚Â³n u orden invÃƒÆ’Ã‚Â¡lidos. El POST devuelve 400 ante entrada invÃƒÆ’Ã‚Â¡lida y 403 si falta pertenencia al ÃƒÆ’Ã‚Â¡rea. Errores internos devuelven 500.






