-- Удаляем индексы из схемы tgbot
DROP INDEX IF EXISTS idx_chat_user_created;
DROP INDEX IF EXISTS idx_chat_user_id;
DROP INDEX IF EXISTS idx_chat_created;

DROP INDEX IF EXISTS idx_summaries_meeting_id;

DROP INDEX IF EXISTS idx_transcriptions_meeting_id;

DROP INDEX IF EXISTS idx_meetings_user_status;
DROP INDEX IF EXISTS idx_meetings_user_created;
DROP INDEX IF EXISTS idx_meetings_status;
DROP INDEX IF EXISTS idx_meetings_user_id;
DROP INDEX IF EXISTS idx_meetings_created;

-- Удаляем таблицы в обратном порядке зависимостей
DROP TABLE IF EXISTS chat_history;
DROP TABLE IF EXISTS summaries;
DROP TABLE IF EXISTS transcriptions;
DROP TABLE IF EXISTS meetings;
DROP TABLE IF EXISTS users;