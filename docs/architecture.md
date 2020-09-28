# MASC - Architecture

Mission Airsoft Control

This paper describes the architecture of Mission Airsoft Control, developed by _TODO_ in Germany.

## Roles

Roles that various devices can take.

- Scheduler: Schedules the different games
- Game Master: Controls the game; usually referred to as "Orga"
- Team Base: Base for a team countdown, interactions and setup. Often merged into one device with player control role.
- Player Control: Allows player interaction like announcing hits or revives.
- Match Stats Collector: Collects various stats and match related information. Application might be a countdown display, current match status or global information.
- Global Information Collector: Collects global information like highscores, current schedule and system status.
- Objective: Supports a _set of game modes_ in which is takes a certain role.