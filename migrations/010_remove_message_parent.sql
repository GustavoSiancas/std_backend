-- message_relations es la unica fuente de verdad para respuestas, reenvios y referencias.
ALTER TABLE messages DROP CONSTRAINT IF EXISTS ck_messages_parent_not_self;
DROP INDEX IF EXISTS idx_messages_parent;
ALTER TABLE messages DROP COLUMN IF EXISTS message_parent_id;
