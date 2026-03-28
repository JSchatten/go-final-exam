-- Удаляем индексы
-- DROP INDEX IF EXISTS idx_chat_created;
-- DROP INDEX IF EXISTS idx_chat_user_id;
-- DROP INDEX IF EXISTS idx_meetings_created;
-- DROP INDEX IF EXISTS idx_meetings_user_id;

-- Удаляем таблицы (в обратном порядке из-за зависимостей)
DROP TABLE IF EXISTS chat_history;
DROP TABLE IF EXISTS summaries;
DROP TABLE IF EXISTS transcriptions;
DROP TABLE IF EXISTS meetings;
DROP TABLE IF EXISTS users;
