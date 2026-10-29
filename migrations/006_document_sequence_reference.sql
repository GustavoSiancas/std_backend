-- Preserve historical formatted numbers, without guessing their reserved value.
ALTER TABLE documents RENAME COLUMN document_number TO legacy_document_number;
ALTER TABLE documents ADD COLUMN document_number bigint REFERENCES document_sequences(id);
ALTER TABLE documents ADD COLUMN correlative_number integer CHECK(correlative_number > 0);
ALTER TABLE documents ADD CONSTRAINT ck_document_sequence_number CHECK (
 (document_number IS NULL AND correlative_number IS NULL) OR
 (document_number IS NOT NULL AND correlative_number IS NOT NULL)
);
CREATE UNIQUE INDEX uq_document_reserved_number ON documents(document_number,correlative_number) WHERE document_number IS NOT NULL;
CREATE UNIQUE INDEX uq_sequence_scope_reference ON document_sequences(id,document_type_id,department_id);
ALTER TABLE documents ADD CONSTRAINT fk_document_sequence_scope FOREIGN KEY(document_number,document_type_id,department_id) REFERENCES document_sequences(id,document_type_id,department_id);


