create table light_switch_assignments
(
    light_switch int not null
        constraint light_switch_assignments_light_switches_id_fk
            references light_switches
            on update cascade on delete cascade,
    fixture      int not null
        constraint light_switch_assignments_fixtures_id_fk
            references fixtures
            on update cascade on delete cascade
);

create unique index light_switch_assignments_light_switch_fixture_uindex
    on light_switch_assignments (light_switch, fixture);

