CREATE TYPE agreement_status AS ENUM (
    'pending', 
    'active',
    'completed',
    'cancelled',
    'defaulted',
    'disputed'
);
CREATE TYPE payment_frequency AS ENUM (
    'one_time',
    'weekly',
    'biweekly',
    'monthly'
);

CREATE TABLE IF NOT EXISTS agreements (
    id SERIAL PRIMARY KEY,
    lender_id INT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    borrower_id INT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    post_id INT NOT NULL REFERENCES posts(id),

    principal_amount NUMERIC(18,2) NOT NULL CHECK (principal_amount > 0),
    interest_rate NUMERIC(5,4) NOT NULL CHECK (interest_rate >= 0),
    total_amount NUMERIC(18,2) NOT NULL CHECK (total_amount >= principal_amount),
    currency CHAR(3) NOT NULL DEFAULT 'KZT',

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    accepted_at TIMESTAMPTZ,
    disbursed_at TIMESTAMPTZ,
    start_date DATE,
    due_date DATE NOT NULL,
    completed_at TIMESTAMPTZ,

    payment_frequency payment_frequency NOT NULL DEFAULT 'one_time',
    number_of_payments INT NOT NULL DEFAULT 1 CHECK (number_of_payments > 0),

    status agreement_status NOT NULL DEFAULT 'pending',
    contract_url TEXT,
    contract_hash TEXT,

    CONSTRAINT valid_parties CHECK (lender_id != borrower_id),
    CONSTRAINT valid_dates CHECK (due_date >= start_date)
);