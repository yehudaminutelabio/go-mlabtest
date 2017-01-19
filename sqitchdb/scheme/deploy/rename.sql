-- Deploy tests:rename to pg

BEGIN;

ALTER TABLE tabel RENAME TO table1;

COMMIT;
