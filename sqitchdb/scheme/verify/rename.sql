-- Verify tests:rename on pg

BEGIN;

SELECT id FROM table1 WHERE FALSE;

ROLLBACK;
