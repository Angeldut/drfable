ALTER TABLE expenses
    ADD COLUMN user_id INTEGER REFERENCES users(id) ON DELETE CASCADE;

CREATE INDEX idx_expenses_user ON expenses(user_id);
