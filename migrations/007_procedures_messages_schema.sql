-- Alinea procedures y messages con el esquema que usa el codigo.
-- Bases creadas con una version anterior tenian department_id / sender_id y no tenian
-- procedure_type ni sender_user_id; CREATE TABLE IF NOT EXISTS de la 001 no las corrigio.
-- Ademas agrega description y procedure_number (correlativo automatico) a procedures.
-- Es idempotente: funciona tanto en bases viejas como en bases nuevas.

-- ===== procedures =====
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='procedures' AND column_name='department_id')
     AND NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='procedures' AND column_name='sender_department_id') THEN
    ALTER TABLE procedures RENAME COLUMN department_id TO sender_department_id;
  END IF;
END $$;

ALTER TABLE procedures ADD COLUMN IF NOT EXISTS sender_user_id bigint REFERENCES users(id);
ALTER TABLE procedures ADD COLUMN IF NOT EXISTS procedure_type varchar(12);
UPDATE procedures SET procedure_type = CASE WHEN sender_department_id IS NULL THEN 'PERSONAL' ELSE 'DEPARTMENT' END WHERE procedure_type IS NULL;
UPDATE procedures SET sender_user_id = created_by WHERE procedure_type = 'PERSONAL' AND sender_user_id IS NULL;
ALTER TABLE procedures ALTER COLUMN procedure_type SET NOT NULL;
ALTER TABLE procedures ALTER COLUMN status SET DEFAULT 'OPEN';

ALTER TABLE procedures DROP CONSTRAINT IF EXISTS ck_procedure_type;
ALTER TABLE procedures ADD CONSTRAINT ck_procedure_type CHECK (procedure_type IN ('PERSONAL','DEPARTMENT'));
ALTER TABLE procedures DROP CONSTRAINT IF EXISTS ck_procedure_sender;
ALTER TABLE procedures ADD CONSTRAINT ck_procedure_sender CHECK (
  (procedure_type = 'PERSONAL' AND sender_user_id IS NOT NULL)
  OR (procedure_type = 'DEPARTMENT' AND sender_user_id IS NULL AND sender_department_id IS NOT NULL));

ALTER TABLE procedures ADD COLUMN IF NOT EXISTS description text;
ALTER TABLE procedures ADD COLUMN IF NOT EXISTS procedure_number varchar(60);
CREATE UNIQUE INDEX IF NOT EXISTS uq_procedures_number ON procedures(procedure_number) WHERE procedure_number IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_procedures_sender_user ON procedures(sender_user_id);
CREATE INDEX IF NOT EXISTS idx_procedures_sender_department ON procedures(sender_department_id);

-- Contador por ambito: 'USER:{id}' para PERSONAL, 'DEPARTMENT:{id}' para DEPARTMENT.
CREATE TABLE IF NOT EXISTS procedure_sequences (
  scope_key varchar(40) PRIMARY KEY,
  last_number integer NOT NULL DEFAULT 0 CHECK (last_number >= 0),
  updated_at timestamptz NOT NULL DEFAULT now()
);

-- ===== messages =====
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='messages' AND column_name='sender_id')
     AND NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='messages' AND column_name='created_by') THEN
    ALTER TABLE messages RENAME COLUMN sender_id TO created_by;
  END IF;
  IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='messages' AND column_name='department_id')
     AND NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='messages' AND column_name='sender_department_id') THEN
    ALTER TABLE messages RENAME COLUMN department_id TO sender_department_id;
  END IF;
END $$;

ALTER TABLE messages ADD COLUMN IF NOT EXISTS sender_user_id bigint REFERENCES users(id);
UPDATE messages SET sender_user_id = created_by WHERE message_type = 'PERSONAL' AND sender_user_id IS NULL;

CREATE INDEX IF NOT EXISTS idx_messages_creator ON messages(created_by);
CREATE INDEX IF NOT EXISTS idx_messages_sender_user ON messages(sender_user_id);
CREATE INDEX IF NOT EXISTS idx_messages_sender_department ON messages(sender_department_id);
CREATE INDEX IF NOT EXISTS idx_messages_procedure ON messages(procedure_id);
