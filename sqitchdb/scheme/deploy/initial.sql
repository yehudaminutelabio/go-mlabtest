-- Deploy tests:initial to pg

BEGIN;

CREATE TABLE tabel (
    id       varchar(15)  PRIMARY KEY,
    name     varchar(256)
);

COMMIT;
