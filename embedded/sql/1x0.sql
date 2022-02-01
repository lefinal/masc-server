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

-- Devices.

create table devices
(
    id               varchar                 not null
        constraint devices_pk
            primary key,
    name             varchar,
    self_description varchar                 not null,
    last_seen        timestamp default now() not null
);

create unique index devices_id_uindex
    on devices (id);

-- Fixtures.

create table if not exists fixtures
(
    id          serial
        constraint fixtures_pk
            primary key,
    device      varchar                 not null
        constraint fixtures_devices_id_fk
            references devices
            on update cascade on delete cascade,
    provider_id varchar                 not null,
    name        varchar,
    type        varchar                 not null,
    last_seen   timestamp default now() not null,
    constraint fixtures_pk_2
        unique (device, provider_id)
);

comment on table fixtures is 'Lighting fixtures';

create unique index if not exists fixtures_id_uindex
    on fixtures (id);
