CREATE TABLE users(
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	name text NOT NULL,
	email text NOT NULL,
	password_hash text NOT NULL,
	currency text NOT NULL DEFAULT 'NGN' CHECK (char_length(currency) = 3),
	created_at timestamptz NOT NULL DEFAULT now(),
	updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX users_email_key ON users (lower(email));