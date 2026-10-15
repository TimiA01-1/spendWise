CREATE TABLE alerts(
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	budget_id uuid NOT NULL REFERENCES budgets(id) ON DELETE CASCADE,
	month date NOT NULL,
	level text NOT NULL CHECK (level IN ('warning', 'limit_reached')),
	amount bigint NOT NULL,
	spent bigint NOT NULL,
	percent_used integer NOT NULL,
	read_at timestamptz,
	created_at timestamptz NOT NULL DEFAULT now(),

	-- The heart of de-duplication: at most ONE alert per budget, month and level.
    UNIQUE (budget_id, month, level)
);

CREATE INDEX alerts_user_created_idx ON alerts (user_id, created_at DESC);