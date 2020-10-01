# MASC - Flows

Mission Airsoft Control

This paper describes common flows in Mission Airsoft Control, developed by _TODO_ in Germany.
All messages have the following structure:
```json
{
  "meta": {
    "type": "the_message_type",
    "device_id": "the_device_id"
  },
  "payload": {}
}
```
If an error occurs during handling messages, or the message contains invalid data, an **error** message with the specific error code:
```json
{
  "error_code": 0,
  "message": "the_error_message"
}
```
Otherwise, an **ok** message is being sent.
If not stated otherwise, **pf** references an performer.

## Gatekeeping
### Login

This describes the login process for all devices.

Flow:

- The new device sends a **hello**-Message with empty device id:

  ```json
  {
    "name": "the_device_name",
    "description": "the_device_description",
    "roles": ["the_role_name"]
  }
  ```
- The _server_ responds with a **welcome**-Message which also contains the assigned device id in the ```meta``` field:
```json
{
  "server_name": "the_server_name"
}
```
From now on the messages sent and received must contain the correct device id.
## Scheduling
Flows for scheduling games and retrieving schedules.
### Retrieve scheduling
Events get retrieved by a _all_ via **get-schedule**:
```json
{}
```
The _server_ responds via **schedule** message:
```json
{
  "events": [
    {
      "id": "the_event_id",
      "type": "match",
      "title": "the_title",
      "description": "the_description",
      "start_time": "the_events_start_time",
      "end_time": "the_events_end_time"
    },
    {}
  ]
}
```
The ```event_id``` for matches are _not_ the same as the match id as they are treated separately for allowing spontaneous matches and so on.
### Post event
Posting an event happens by _scheduler_ via ***schedule-event*** message:
```json
{
  "event": {
    "type": "game",
    "title": "the_title",
    "description": "the_description",
    "start_time": "the_events_start_time",
    "end_time": "the_events_end_time"
  }
}
```
### Update event
Updating an event happens by _scheduler_ via ***update-event** message:
```json
{
  "event": {}
}
```
### Delete event
Deleting an event happens by _scheduler_ via **delete-event** message:
```json
{
  "event_id": "the_event_id"
}
```
## Games
All stuff related to games. Should follow mainly the order of the chapters here.
### Set up
Each match has to be setup although default configurations should be available.
The _game master_ creates a new match via **new-match** message:

```json
{}
```
The _server_ responds with a **request-game-mode** message:
```json
{
  "match_id": "the_assigned_match_id",
  "offered_game_modes": []
}
```
The _game master_ then chooses a game mode and tells the _server_ by **set-game-mode** message:
```json
{
  "match_id": "the_match_id",
  "game_mode": "the_chosen_game_mode"
}
```
#### Match config

The _server_ now sends a **match-config** message:

```json
{
  "match_id": "the_match_id",
  "game_mode": "the_set_game_mode",
  "team_configs": [],
  "match_config": {}
}
```
The structure of the match config may vary from game mode to game mode.
The _game master_ sets the match config via **setup-match** message:

```json
{
  "match_id": "the_match_id",
  "team_configs": {},
  "match_config": {}
}
```
If the _game master_ wants to request match config presets, he can do so via **request-match-config-presets** message:
```json
{
  "game_mode": "the_target_game_mode"
}
```
The response will be a **match-config-presets** message:
```json
{
  "game_mode": "the_target_game_mode",
  "presets": []
}
```
The match config is confirmed by the _game master_ via **confirm-match-config** message:
```json
{
  "match_id": "the_match_id"
}
```
#### Role assignment

The _server_ requests the role assignment by sending a **request-role-assignment** message:

```json
{
  "match_id": "the_match_id",
  "roles": [
    {
      "performer_id": "the_performer_id",
      "role_details": {
        "role": "the_role",
        "name": "the_role_name_for_this_particular_match",
        "description": "description_for_usage_for_this_particular_match"
      },
      "available_devices": [
         {
           "id": "the_device_id",
           "name": "the_device_name",
           "description": "the_device_description"
         }
      ]
    }
  ]
}
```
Assigning happens by the _game master_ via **assign-roles** message:
```json
{
  "match_id": "the_match_id",
  "role_assignments": [
    {
      "performer_id": "the_performer_id",
      "device_id": "the_device_id"
    }
  ]
}
```
The _server_ then sends to the corresponding devices a **your-in** message:
```json
{
  "match_id": "the_match_id",
  "team_config": {},
  "match_config": {},
  "contracts": [
      {
          "performer_id": "the_assigned_performer_id",
          "role_details": {
              "role": "the_role_key",
              "name": "the_role_name",
              "description": "the_role_description"
          }
      }
  ]
}
```
The _server_ then allows player login via **player-login-status** message:
```json
{
  "match_id": "the_match_id",
  "player-login-open": true,
  "teams": [
    {
      "team_config": {},
      "players":  [
        {
          "user_id": "the_user_id"
        }
      ] 
    }
  ]
}
```
Player login happens by _player controls_ via **login-player** message:
```json
{
  "match_id": "the_match_id",
  "user_id": "the_user_id",
  "team_id": "the_team_id"
}
```
Performers should not request configs as everything necessary should be shipped with the match config.

Each performer, in the match involved, sends a **ready-for-match-start** message if it is ready for each role! Currently, this cannot be undone.
```json
{
  "match_id": "the_match_id",
  "role_id": "the_role_id"
}
```
After all performers have told that they are ready, the _server_ sends one final **player-login-status** message (see above) with adjusted player login open indicator. The _server_ then sends a **prepare-for-match-start** message which allows for example dimming a team display or an intro for the countdown display after that it waits for a certain time (set in server config):
```json
{
  "match_id": "the_match_id"
}
```
