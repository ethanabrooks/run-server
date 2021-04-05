BEGIN;
CREATE TYPE method AS ENUM ('grid', 'random');
CREATE TABLE metadata (
    username text,
    project text
);

CREATE TABLE sweep (
    ID serial primary key,
    Method   method not null,
    GridIndex int,
    "description" text
) INHERITS (metadata);

CREATE TABLE run (
    ID serial primary key,
    SweepID int references sweep(id),
    CommitHash text not null,
    Command text not null,
    Parameters json,
    "description" text
) INHERITS (metadata);

CREATE TABLE sweep_parameter (
    SweepID   integer not null references sweep(ID),
    "Key"     text not null,
    "Values"  json[] not null,
    unique (SweepID, "Key")
);

CREATE TABLE run_log (
    ID serial primary key,
    RunId int not null references run(id),
    Document json not null
);
COMMIT;
