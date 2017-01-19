-- Revert tests:rename from pg

BEGIN;

ALTER TABLE tabel1 RENAME TO tabel;

COMMIT;
