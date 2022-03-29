create table devices
(
    id        varchar not null
        constraint devices_pk
            primary key,
    type      varchar not null,
    name      varchar,
    last_seen timestamp,
    config    jsonb
);

comment on table devices is 'Collection of known devices.';

comment on column devices.id is 'Device ID that is randomly chosen by devices.';

comment on column devices.type is 'Type of the device that is announced by the device itself.';

comment on column devices.name is 'Human-readable name.';

comment on column devices.config is 'Optional configuration for the device.';

create unique index devices_id_uindex
    on devices (id);

-- Increment version.

update masc
set value='2'
where key = 'db-version';