CREATE TABLE budgets(
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	category_id uuid REFERENCES categories(id),
	amount bigint NOT NULL CHECK (amount > 0),
	alert_threshold integer NOT NULL DEFAULT 80 CHECK(alert_threshold BETWEEN 1 AND 100),
	created_at timestamptz NOT NULL DEFAULT now(),
	updated_at timestamptz NOT NULL DEFAULT now()
);

-- One budget per category per user, and at most one overall budget per user.
CREATE UNIQUE INDEX budgets_user_category_key ON budgets (user_id, category_id) WHERE category_id IS NOT NULL;
CREATE UNIQUE INDEX budgets_user_overall_key  ON budgets (user_id) WHERE category_id IS NULL;