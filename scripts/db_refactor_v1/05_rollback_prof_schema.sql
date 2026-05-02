PRAGMA foreign_keys = OFF;
BEGIN;

DROP TABLE IF EXISTS manual_notifications_v2;
DROP TABLE IF EXISTS reservations_v2;
DROP TABLE IF EXISTS requests_v2;
DROP TABLE IF EXISTS vendor_identity_map_v2;
DROP TABLE IF EXISTS api_accounts_v2;
DROP TABLE IF EXISTS users_v2;

COMMIT;
PRAGMA foreign_keys = ON;
