CREATE TYPE notification_type AS ENUM (
    'agreement_created',
    'agreement_accepted',
    'agreement_cancelled',
    'payment_reminder',
    'payment_overdue',
    'agreement_completed'
);

CREATE TYPE notification_status AS ENUM (
    'unread',
    'read',
    'archived'
);

CREATE TABLE IF NOT EXISTS notifications (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type notification_type NOT NULL,
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    status notification_status NOT NULL DEFAULT 'unread',
    
    -- metadata for linking to related entities
    agreement_id INT REFERENCES agreements(id) ON DELETE SET NULL,
    
    -- additional metadata as JSON
    metadata JSONB,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    read_at TIMESTAMPTZ,
    
    -- index for quick user queries
    INDEX idx_notifications_user_id (user_id),
    INDEX idx_notifications_user_status (user_id, status),
    INDEX idx_notifications_created_at (created_at DESC)
);

