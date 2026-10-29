-- Estado de entrega de los mensajes. Los existentes se consideran enviados.
-- El valor por defecto es BORRADOR para que una insercion incompleta nunca se publique.
ALTER TABLE messages ADD COLUMN IF NOT EXISTS status varchar(10) NOT NULL DEFAULT 'BORRADOR';
UPDATE messages SET status = 'HECHO' WHERE status = 'BORRADOR';
UPDATE messages SET sent_at = created_at WHERE status = 'HECHO' AND sent_at IS NULL;

ALTER TABLE messages DROP CONSTRAINT IF EXISTS ck_messages_status;
ALTER TABLE messages ADD CONSTRAINT ck_messages_status CHECK (status IN ('BORRADOR', 'HECHO'));

-- Un borrador no se ha enviado; un mensaje hecho/enviado siempre conserva su fecha.
ALTER TABLE messages DROP CONSTRAINT IF EXISTS ck_messages_status_sent_at;
ALTER TABLE messages ADD CONSTRAINT ck_messages_status_sent_at CHECK (
  (status = 'BORRADOR' AND sent_at IS NULL)
  OR (status = 'HECHO' AND sent_at IS NOT NULL)
);

CREATE INDEX IF NOT EXISTS idx_messages_creator_status ON messages(created_by, status);
