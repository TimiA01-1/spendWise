-- name: ListCategoriesForUser :many
SELECT id, name, type, (user_id IS NULL)::boolean AS is_default, created_at
FROM categories
WHERE user_id IS NULL OR user_id = sqlc.arg(user_id)::uuid
ORDER BY type, (user_id IS NULL) DESC, lower(name);

-- name: CreateCategory :one
INSERT INTO categories (user_id, name, type)
SELECT sqlc.arg(user_id)::uuid, sqlc.arg(name)::text, sqlc.arg(type)::text
WHERE NOT EXISTS (
    SELECT 1 FROM categories
    WHERE user_id IS NULL
      AND lower(name) = lower(sqlc.arg(name)::text)
      AND type = sqlc.arg(type)::text
)
RETURNING id, name, type, false::boolean AS is_default, created_at;