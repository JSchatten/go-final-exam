-- -- Создаём схему tgbot, если её нет
-- CREATE SCHEMA IF NOT EXISTS tgbot;

-- Обеспечиваем наличие pgcrypto для gen_random_uuid()
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- 1. Пользователи Telegram
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    telegram_id BIGINT UNIQUE NOT NULL,
    username VARCHAR(255),
    first_name VARCHAR(255),
    last_name VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- 2. Встречи (аудиофайлы, загруженные пользователями)
CREATE TABLE meetings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id BIGINT REFERENCES users(telegram_id) ON DELETE CASCADE,
    title VARCHAR(512) DEFAULT 'Meeting',
    audio_file_path TEXT,        -- путь или URL к сохранённому файлу
    status VARCHAR(50) DEFAULT 'uploaded', -- uploaded, processing, completed, failed
    error_message TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 3. Транскрипция (результат SaluteSpeech)
CREATE TABLE transcriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    meeting_id UUID REFERENCES meetings(id) ON DELETE CASCADE,
    salute_task_id VARCHAR(50),
    status VARCHAR(50) DEFAULT 'NONE', -- NEW, RUNNING, CANCELED, DONE, ERROR
    full_text TEXT,
    processed_at TIMESTAMPTZ DEFAULT NOW()
);

-- 4. Краткое содержание (результат GigaChat)
CREATE TABLE summaries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    meeting_id UUID REFERENCES meetings(id) ON DELETE CASCADE,
    summary_text TEXT NOT NULL,
    generated_at TIMESTAMPTZ DEFAULT NOW()
);

-- 5. История взаимодействий с GigaChat (команда /chat)
CREATE TABLE chat_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id BIGINT REFERENCES users(telegram_id) ON DELETE CASCADE,
    query_text TEXT NOT NULL,
    response_text TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);


-- Индексы

-- Для встреч
CREATE INDEX idx_meetings_user_id ON meetings(user_id);
CREATE INDEX idx_meetings_created ON meetings(created_at);
CREATE INDEX idx_meetings_status ON meetings(status); -- фильтрация по статусу (processing, completed)
CREATE INDEX idx_meetings_user_status ON meetings(user_id, status); -- для списка встреч пользователя с фильтром
CREATE INDEX idx_meetings_user_created ON meetings(user_id, created_at DESC); -- для сортировки "сначала новые"

-- Для транскрипций
CREATE INDEX idx_transcriptions_meeting_id ON transcriptions(meeting_id); -- JOIN с meetings

-- Для выжимок
CREATE INDEX idx_summaries_meeting_id ON summaries(meeting_id); -- JOIN с meetings

-- Для истории чата
CREATE INDEX idx_chat_user_id ON chat_history(user_id);
CREATE INDEX idx_chat_created ON chat_history(created_at DESC);
CREATE INDEX idx_chat_user_created ON chat_history(user_id, created_at DESC); -- последние запросы пользователя

-- Для индексации текста
/*
CREATE EXTENSION IF NOT EXISTS pg_trgm;
-- Для заголовков встреч
CREATE INDEX idx_meetings_title_trgm ON meetings USING gin (title gin_trgm_ops);
-- Для транскрипций
CREATE INDEX idx_transcriptions_full_text_trgm ON transcriptions USING gin (full_text gin_trgm_ops);
-- Для выжимок
CREATE INDEX idx_summaries_summary_text_trgm ON summaries USING gin (summary_text gin_trgm_ops);
*/