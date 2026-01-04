-- 1. Merge duplicates: keeping the one with the latest last_message_at
WITH duplicates AS (
    SELECT id, 
           ROW_NUMBER() OVER (
               PARTITION BY tenant_id, contact_number 
               ORDER BY last_message_at DESC NULLS LAST, created_at DESC
           ) as rn
    FROM conversations
)
DELETE FROM conversations
WHERE id IN (SELECT id FROM duplicates WHERE rn > 1);

-- 2. Add Unique Constraint
CREATE UNIQUE INDEX IF NOT EXISTS conversations_tenant_contact_idx 
ON conversations (tenant_id, contact_number);
