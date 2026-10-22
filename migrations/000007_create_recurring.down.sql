DROP INDEX IF EXISTS transactions_recurring_once_key;
ALTER TABLE transactions DROP COLUMN IF EXISTS recurring_for, DROP COLUMN IF EXISTS recurring_id;
DROP TABLE IF EXISTS recurring_transactions;