# CorrecciÃ³n: IDs numÃ©ricos autogenerados

## Cambio de contrato

STDB2 ya no expone ni acepta UUIDs. Todas las entidades usan un `id` numÃ©rico positivo (`bigint`) asignado por PostgreSQL mediante identidad. En el frontend los IDs se manejan como nÃºmeros JSON, no como strings.

Los campos de relaciÃ³n â€”por ejemplo `department_id`, `document_type_id`, `created_by`, `document_ids` y los parÃ¡metros de ruta `:id`â€” tambiÃ©n son enteros positivos.

## Endpoints existentes

No cambian las rutas. Cambia solamente el tipo de los valores:

- `GET /api/v1/documents/42`
- `GET /api/v1/documents?department_id=7&document_type_id=3`
- `POST /api/v1/users` con `"department_id": 7`
- `POST /api/v1/documents` con `"document_type_id": 3`, `"department_id": 7` y `"created_by": 12`

Un UUID, un string no numÃ©rico, cero o un nÃºmero negativo se rechaza con HTTP 400 cuando el endpoint valida el identificador. Los recursos que no existen conservan HTTP 404.

## Numeración visible de documentos

El consecutivo interno aumenta como `1`, `2`, `3` para cada ámbito de secuencia. Para documentos con correlativo, `document_number` es el código visible con este formato: `STD-<consecutivo>-<código_del_departamento>`. Ejemplo: el primer documento de Sistemas cuyo código de departamento es `CIS` devuelve `"document_number": "STD-1-CIS"`; el siguiente devuelve `STD-2-CIS`.

`document_sequence_id` es el ID numérico interno del contador y solo se incluye como referencia técnica. No debe usarse para mostrar ni construir el número del documento. Los documentos sin correlativo devuelven `document_number`, `document_sequence_id` y `correlative_number` como `null`.

## Base de datos y despliegue

Las instalaciones nuevas crean PK y FK `bigint` con IDs autogenerados. No se debe editar una migraciÃ³n que ya fue registrada en una base instalada: una base histÃ³rica con UUID requiere una migraciÃ³n de datos planificada, con respaldo, porque sus PK y todas sus FK deben traducirse de forma consistente. No es seguro convertir UUID a entero con un cast.

Antes de desplegar, ejecute las pruebas y aplique las migraciones en un entorno de prueba.



