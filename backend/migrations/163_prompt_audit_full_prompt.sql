-- Full prompt text is retained only on review events and is returned by detail APIs only.
ALTER TABLE prompt_audit_events
    ADD COLUMN IF NOT EXISTS full_prompt TEXT NOT NULL DEFAULT '';
