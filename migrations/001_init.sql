-- Задачи
CREATE TABLE IF NOT EXISTS situations (
    id SERIAL PRIMARY KEY,
    answer TEXT NOT NULL,
    is_used BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Фотографии
CREATE TABLE IF NOT EXISTS photos (
    id SERIAL PRIMARY KEY,
    situation_id INTEGER NOT NULL REFERENCES situations(id) ON DELETE CASCADE,
    file_id TEXT NOT NULL,
    sort_order INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_photos_situation_id ON photos(situation_id);