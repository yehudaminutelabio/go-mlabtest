-- Verify tests:initial on pg

BEGIN;

SELECT id FROM tabel WHERE FALSE;

ROLLBACK;
