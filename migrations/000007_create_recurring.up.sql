CREATE TABLE recurring_transactions(
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	-- No ON DELETE on these two: an account or category that a rule still uses cant be deleted
	account_id uuid NOT NULL REFERENCES accounts(id),
	category_id uuid NOT NULL REFERENCES categories(id),
	type text NOT NULL CHECK (type IN ('income', 'expense')),
	amount bigint NOT NULL CHECK (amount > 0),
	note text NOT NULL DEFAULT '',
	frequency text NOT NULL CHECK (frequency IN ('daily', 'weekly', 'monthly', 'yearly')),
	start_date date NOT NULL,
	end_date date,
	next_index integer NOT NULL DEFAULT 0,
	next_run_on date NOT NULL,
	status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'paused', 'finished')),
	runs integer NOT NULL DEFAULT 0,
	last_error text NOT NULL DEFAULT '',
	retry_at timestamptz NOT NULL DEFAULT now(),
	created_at timestamptz NOT NULL DEFAULT now(),
	updated_at timestamptz NOT NULL DEFAULT now(),
	CHECK (end_date IS NULL OR end_date >= start_date)
);

CREATE INDEX recurring_user_idx ON recurring_transactions (user_id);

CREATE INDEX recurring_due_idx ON recurring_transactions (next_run_on) WHERE status = 'active';

-- Remember which rule and which occurrence created a transaction.
ALTER TABLE transactions
    ADD COLUMN recurring_id  uuid REFERENCES recurring_transactions(id) ON DELETE SET NULL,
    ADD COLUMN recurring_for date;

-- Safety net: one rule can create at most ONE transaction per occurrence date.
CREATE UNIQUE INDEX transactions_recurring_once_key
    ON transactions (recurring_id, recurring_for) WHERE recurring_id IS NOT NULL;