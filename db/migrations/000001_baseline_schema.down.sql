BEGIN;

DROP TABLE IF EXISTS password_reset_tokens;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS notifications;
DROP TABLE IF EXISTS service_requests;
DROP TABLE IF EXISTS doers;
DROP TABLE IF EXISTS customers;

COMMIT;