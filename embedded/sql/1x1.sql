-- auto-generated definition
create table mqtt_devices
(
    mqtt_id   varchar not null
        constraint mqtt_devices_pk
            primary key,
    device_id varchar not null
);

create unique index mqtt_devices_mqtt_id_uindex
    on mqtt_devices (mqtt_id);

-- auto-generated definition
create table light_switches
(
    id          serial
        constraint light_switches_pk
            primary key,
    device      varchar                 not null,
    provider_id varchar                 not null,
    name        varchar,
    type        varchar                 not null,
    last_seen   timestamp default now() not null
);

create unique index light_switches_id_uindex
    on light_switches (id);
