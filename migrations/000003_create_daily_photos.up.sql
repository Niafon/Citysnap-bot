CREATE TABLE daily_photos (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID REFERENCES users(id) ON DELETE CASCADE,
    city          VARCHAR(100) NOT NULL,
    photo_file_id VARCHAR(255) NOT NULL,
    caption       VARCHAR(500) DEFAULT '',
    view_count    INT DEFAULT 0,
    created_at    TIMESTAMPTZ DEFAULT now(),
    expires_at    TIMESTAMPTZ NOT NULL,
    is_visible    BOOLEAN DEFAULT true
);

CREATE INDEX idx_photos_feed ON daily_photos(city, is_visible, created_at DESC) WHERE is_visible = true;
