CREATE TABLE swipes (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    swiper_id  UUID REFERENCES users(id) ON DELETE CASCADE,
    swiped_id  UUID REFERENCES users(id) ON DELETE CASCADE,
    type       VARCHAR(16) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now(),
    UNIQUE(swiper_id, swiped_id)
);
