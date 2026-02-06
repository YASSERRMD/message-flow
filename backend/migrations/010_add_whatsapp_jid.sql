ALTER TABLE conversations ADD COLUMN IF NOT EXISTS whatsapp_jid TEXT;

-- Backfill best-effort for existing rows.
UPDATE conversations
SET whatsapp_jid = CASE
    WHEN contact_number LIKE '%@%' THEN contact_number
    WHEN contact_number LIKE '12036%' THEN contact_number || '@g.us'
    ELSE contact_number || '@s.whatsapp.net'
END
WHERE whatsapp_jid IS NULL OR whatsapp_jid = '';

CREATE INDEX IF NOT EXISTS conversations_tenant_whatsapp_jid_idx
ON conversations (tenant_id, whatsapp_jid);

