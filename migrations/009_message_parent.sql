-- Permite navegar directamente desde una respuesta hasta el mensaje respondido.
ALTER TABLE messages ADD COLUMN IF NOT EXISTS message_parent_id bigint REFERENCES messages(id);

-- Conserva el vinculo de las respuestas creadas antes de esta columna.
UPDATE messages m
SET message_parent_id = relations.target_message_id
FROM (
  SELECT DISTINCT ON (source_message_id) source_message_id, target_message_id
  FROM message_relations
  WHERE relation_type = 'REPLY'
  ORDER BY source_message_id, id
) relations
WHERE m.id = relations.source_message_id
  AND m.message_parent_id IS NULL;

ALTER TABLE messages DROP CONSTRAINT IF EXISTS ck_messages_parent_not_self;
ALTER TABLE messages ADD CONSTRAINT ck_messages_parent_not_self
  CHECK (message_parent_id IS NULL OR message_parent_id <> id);

CREATE INDEX IF NOT EXISTS idx_messages_parent ON messages(message_parent_id);
