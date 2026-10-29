# Correlativos por tipo documental y Ã¡rea

Se reutiliza document_sequences: no se crea una tabla duplicada.

| Campo | FunciÃ³n |
| --- | --- |
| id | IDs numéricos autogenerados del registro contador |
| document_type_id | IDs numéricos autogenerados del tipo documental |
| department_id | IDs numéricos autogenerados del Ã¡rea; null para Ã¡mbito personal/global |
| year | AÃ±o de numeraciÃ³n |
| last_number | Ãšltimo correlativo reservado |

Ejemplo del mismo aÃ±o: Carta de Sistemas 1, Carta de Sistemas 2, Carta de Contabilidad 1, Oficio de Sistemas 1. Cada combinaciÃ³n tipo + Ã¡rea + aÃ±o tiene su propia fila y contador. Los IDs numéricos autogenerados identifican las entidades; el nÃºmero que aumenta es last_number.

Se mantiene el Ã¡mbito anual de la especificaciÃ³n original: un nuevo aÃ±o comienza en 1. La fila se crea al emitir el primer documento; no es necesario inicializar todas las combinaciones.

El incremento y la creaciÃ³n del documento ocurren en la misma transacciÃ³n. Si falla, se revierte la reserva. El bloqueo y el incremento atÃ³mico evitan duplicados entre solicitudes concurrentes, incluso en el primer uso del contador.

El nÃºmero documental incluye cÃ³digo del tipo, secuencia, aÃ±o e ID de Ã¡rea para distinguir nÃºmeros iguales emitidos por Ã¡reas diferentes. Sin Ã¡rea usa GLOBAL. No se renumeran documentos anteriores.

La migraciÃ³n 005 registra Carta, Oficio, Informe, MemorÃ¡ndum, MemorÃ¡ndum mÃºltiple y Carta mÃºltiple con has_correlative=true. Los tipos sin correlativo conservan document_number=null. Las migraciones se aplican automÃ¡ticamente al iniciar.


