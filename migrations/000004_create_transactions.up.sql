CREATE TABLE transactions(
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	user_id uuid NOT NULL REFERENCES users(ID) ON DELETE CASCADE,
	-- no ON DELETE clause: the database will not delete an account or category that still has transactions
	account_id uuid NOT NULL REFERENCES accounts(id),
	category_id uuid NOT NULL REFERENCES categories(id),
	type text NOT NULL CHECK (type IN ('income', 'expense')),
	amount bigint NOT NULL CHECK (amount > 0),
	note text NOT NULL DEFAULT '',
	occurred_on date NOT NULL,
	created_at timestamptz NOT NULL DEFAULT now(),
	updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX transactions_user_date_idx ON transactions (user_id, occurred_on DESC, created_at DESC);
CREATE INDEX transactions_account_idx ON transactions (account_id);
CREATE INDEX transactions_category_idx ON transactions (category_id);