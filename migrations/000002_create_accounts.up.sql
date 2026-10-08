CREATE TABLE accounts(
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	user_id uuid NOT NULl REFERENCES users(id) ON DELETE CASCADE,
	name text NOT NULL,
	type text NOT NULL CHECK (type IN ('bank', 'cash', 'wallet')),
	currency text NOT NULL DEFAULT 'NGN' CHECK (char_length(currency) = 3),
	balance bigint NOT NULL DEFAULT 0,
	created_at timestamptz NOT NULL DEFAULT now(),
	updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX accounts_user_id_idx ON accounts (user_id);

-- a user cannot have two accounts with the same name(case sensitive) --
CREATE UNIQUE INDEX accounts_user_name_key ON accounts (user_id, lower(name));