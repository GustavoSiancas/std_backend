-- Each type is a normal relational record with its own numeric ID.
INSERT INTO document_types(name,code,has_correlative,is_active) VALUES
 ('Carta','CARTA',true,true),
 ('Oficio','OFICIO',true,true),
 ('Informe','INFORME',true,true),
 ('MemorÃ¡ndum','MEMORANDUM',true,true),
 ('MemorÃ¡ndum mÃºltiple','MEMORANDUM_MULTIPLE',true,true),
 ('Carta mÃºltiple','CARTA_MULTIPLE',true,true)
ON CONFLICT (code) DO UPDATE SET has_correlative=true;

ALTER TABLE document_sequences ADD CONSTRAINT ck_sequence_nonnegative CHECK(last_number >= 0);


