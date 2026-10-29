# Crear documento

POST http://localhost:8080/api/v1/documents

No requiere token. El frontend envÃƒÆ’Ã‚Â­a created_by con el ID recibido en login. Este dato se valida contra un usuario activo, pero no demuestra identidad mientras no exista autenticaciÃƒÆ’Ã‚Â³n.

Header: Content-Type: application/json

```json
{
  "document_type_id": 3,
  "department_id": 7,
  "created_by": 12,
  "name": "Solicitud de informaciÃƒÆ’Ã‚Â³n",
  "description": "DescripciÃƒÆ’Ã‚Â³n opcional",
  "file_url": "https://example.com/documentos/carta.pdf"
}
```

Todos los campos son obligatorios salvo description. El usuario debe tener pertenencia activa al ÃƒÆ’Ã‚Â¡rea. El ÃƒÆ’Ã‚Â¡rea, usuario y tipo deben estar activos. Cualquier miembro puede crear: no requiere admin ni rol MANAGER.

La URL debe ser absoluta HTTP/HTTPS, sin usuario/contraseÃƒÆ’Ã‚Â±a incrustados. Se guarda el enlace sin descargar el PDF ni verificar su contenido; puede ser un enlace firmado sin extensiÃƒÆ’Ã‚Â³n .pdf. No se implementa subida de archivos.

Respuesta 201 (IDs ilustrativos):

```json
{
  "id": 42,
  "created_at": "2026-10-28T20:00:00Z",
  "updated_at": "2026-10-28T20:00:00Z",
  "document_type_id": 3,
  "department_id": 7,
  "created_by": 12,
  "document_number": "STD-1-SIS",
  "document_sequence_id": 18,
  "correlative_number": 1,
  "name": "Solicitud de informaciÃƒÆ’Ã‚Â³n",
  "description": "DescripciÃƒÆ’Ã‚Â³n opcional",
  "file_url": "https://example.com/documentos/carta.pdf"
}
```

document_number es el código visible STD-<consecutivo>-<código de departamento>; document_sequence_id es la referencia numérica interna al contador. Se comparte entre documentos del mismo ÃƒÆ’Ã‚Â¡mbito. correlative_number conserva el nÃƒÆ’Ã‚Âºmero reservado para ese documento: 1, 2, 3... La pareja es ÃƒÆ’Ã‚Âºnica. El aÃƒÆ’Ã‚Â±o corresponde a America/Bogota y estÃƒÆ’Ã‚Â¡ guardado en document_sequences. Para tipos sin correlativo, ambos campos son null.

La creaciÃƒÆ’Ã‚Â³n del contador, incremento, documento y auditorÃƒÆ’Ã‚Â­a se ejecutan en una sola transacciÃƒÆ’Ã‚Â³n.

GET /api/v1/document-types devuelve data con tipos (id, name, code, has_correlative, is_active), sin token. Para el formulario use solo tipos activos. Las ÃƒÆ’Ã‚Â¡reas se obtienen con GET /api/v1/departments?is_active=true.

Errores: 400 para entrada invÃƒÆ’Ã‚Â¡lida o referencias inexistentes/inactivas; 403 si el usuario no pertenece al ÃƒÆ’Ã‚Â¡rea; 500 para errores internos.

La migraciÃƒÆ’Ã‚Â³n 006 corre automÃƒÆ’Ã‚Â¡ticamente. Conserva nÃƒÆ’Ã‚Âºmeros formateados antiguos en legacy_document_number, sin inferir ni renumerar sus correlativos. Los documentos nuevos usan document_number es el código visible; document_sequence_id es el ID interno del contador.






