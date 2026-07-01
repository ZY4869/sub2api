-- Store the normalized local rule keyword that triggered moderation.

ALTER TABLE content_moderation_audits
  ADD COLUMN IF NOT EXISTS matched_keyword TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_content_moderation_audits_matched_keyword
  ON content_moderation_audits (matched_keyword)
  WHERE matched_keyword <> '';
