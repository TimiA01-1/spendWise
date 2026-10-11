CREATE TABLE categories(
	id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	user_id uuid REFERENCES users(id) ON DELETE CASCADE,
	name text NOT NULL,
	type text NOT NULL CHECK(type IN('income', 'expense')),
	created_at timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX categories_default_name_key ON categories (lower(name), type) WHERE user_id IS NULL;
CREATE UNIQUE INDEX categories_user_name_key ON categories (user_id, lower(name), type) WHERE user_id IS NOT NULL;

INSERT INTO categories (name, type) VALUES
	('Food', 'expense'),
	('Transport', 'expense'),
	('Bills', 'expense'),
	('Shopping', 'expense'),
	('Entertainment', 'expense'),
	('Health', 'expense'),
	('Education', 'expense'),
	('Other', 'expense'),
	('Salary', 'income'),
	('Business', 'income'),
	('Gift', 'income'),
	('Other', 'income');