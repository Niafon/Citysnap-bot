CREATE TABLE users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    telegram_id   BIGINT UNIQUE NOT NULL,
    nickname      VARCHAR(64) NOT NULL DEFAULT '',
    age           INT DEFAULT 0,
    description   TEXT DEFAULT '',
    photo_file_id VARCHAR(255) DEFAULT '',
    city          VARCHAR(100) DEFAULT '',
    is_active     BOOLEAN DEFAULT true,
    created_at    TIMESTAMPTZ DEFAULT now(),
    updated_at    TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX idx_users_city_active ON users(city, is_active) WHERE is_active = true;
