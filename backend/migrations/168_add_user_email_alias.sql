ALTER TABLE users
  ADD COLUMN IF NOT EXISTS email_alias VARCHAR(255) NOT NULL DEFAULT '';

WITH normalized AS (
    SELECT
        id,
        CASE
            WHEN split_part(lower(trim(email)), '@', 2) IN ('gmail.com', 'googlemail.com') THEN
                regexp_replace(split_part(split_part(lower(trim(email)), '@', 1), '+', 1), '\.', '', 'g') || '@gmail.com'
            ELSE
                split_part(lower(trim(email)), '+', 1) || '@' || split_part(lower(trim(email)), '@', 2)
        END AS alias
    FROM users
    WHERE email_alias = ''
),
ranked AS (
    SELECT
        id,
        alias,
        row_number() OVER (PARTITION BY alias ORDER BY id ASC) AS alias_rank
    FROM normalized
)
UPDATE users u
SET email_alias = CASE
    WHEN ranked.alias = '@' THEN lower(trim(u.email))
    WHEN ranked.alias_rank = 1 THEN ranked.alias
    ELSE ranked.alias || '#legacy-' || u.id::text
END
FROM ranked
WHERE u.id = ranked.id;

CREATE UNIQUE INDEX IF NOT EXISTS users_email_alias_unique_active
    ON users (email_alias)
    WHERE deleted_at IS NULL AND email_alias <> '';
