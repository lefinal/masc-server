-- MASC key-val.

create table masc
(
    key   varchar not null
        constraint masc_pk
            primary key,
    value varchar not null
);

create unique index masc_key_uindex
    on masc (key);