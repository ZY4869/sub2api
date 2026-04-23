ALTER TABLE groups
ADD COLUMN IF NOT EXISTS image_protocol_mode VARCHAR(20) NOT NULL DEFAULT 'inherit';

UPDATE groups
SET image_protocol_mode = 'inherit'
WHERE image_protocol_mode IS NULL
   OR btrim(image_protocol_mode) = '';
