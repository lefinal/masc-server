# MASC - Flows

Mission Airsoft Control

This paper describes common flows in Mission Airsoft Control, developed by _TODO_ in Germany.
All messages include the following field ```"meta"``` with the message type in lower case and with hyphens instead of spaces:
```json
{
  "type": "the_message_type",
  "id": "the_sender_id"
}
```
If an error occurs during handling messages, or the message contains invalid data, an **error** message with the specific error code:
```json
{
  "error_code": 0,
  "message": "the_error_message"
}
``` 
Otherwise an **ok** message is being sent.
If not stated otherwise, **dvc** references the client.

## Gatekeeping
### Login

This describes the login process for all devices.

Referring:

- **nd** as the device which wants to log in
- **srv** as the server

Flow:

- **nd** sends a **hello**-Message:

  ```json
  {
    "name": "the_device_name",
    "roles": []
  }
  ```
- The **srv** responds with a **welcome**-Message which also contains the assigned id in the ```meta``` field:
```json
{
  "server_name": "the_server_name"
}
```

## Scheduling
Flows for scheduling games and retrieving schedules.
### Retrieve scheduling
Events get retrieved by the **dvc** via **get-schedule**:
```json
{}
```
The **srv** responds via **schedule** message:
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
Posting an event happens by **dvc** via ***schedule-event*** message:
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
Updating an event happens by **dvc** via ***update-event** message:
```json
{
  "event": {}
}
```
### Delete event
Deleting an event happens by **dvc** via **delete-event** message:
```json
{
  "event_id": "the_event_id"
}
```
## Games
All stuff related to games. Should follow mainly the order of the chapters here.
### Set up
Each match has to be setup although default configurations should be available.
The **dvc** creates a new match via **new-match** message:
```json
{}
```
The **srv** responds with a **request-game-mode** message:
```json
{
  "match_id": "the_assigned_match_id",
  "offered_game_modes": []
}
```
The **dvc** then chooses a game mode and tells the **srv** by **set-game-mode** message:
```json
{
  "match_id": "the_match_id",
  "game_mode": "the_chosen_game_mode"
}
```
The **srv** responds with a **match-config** message:
```json
{
  "match_id": "the_match_id",
  "game_mode": "the_set_game_mode",
  "team_configs": [],
  "match_config": {}
}
```
The structure of the match config may vary from game mode to game mode.
The **dvc** sets the match config via **setup-match** message:
```json
{
  "match_id": "the_match_id",
  "team_configs": {},
  "match_config": {}
}
```
If the **dvc** wants to request prebuild presets, he can do so via **request-match-config-presets** message:
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
The match config is confirmed by the **dvc** via **confirm-match-config** message:
```json
{
  "match_id": "the_match_id"
}
```
The **srv** then requests the role assignment by sending a **request-role-assignment** message:
```json
{
  "match_id": "the_match_id",
  "roles": [
    {
      "role": "the_role",
      "description": "description_for_usage_for_this_particular_match",
      "count_required": 0,
      "assigned_devices": [],
      "available_devices": [
        {
          "id": "the_device_id",
          "name": "the_device_name",
          "description": "the_device_description",
          "roles": []
        }
      ]
    }
  ]
}
```
Assigning happens by the game master via **assign-role** message:
```json
{
  "match_id": 
}
```
The **srv** then allows player login via **player-login-status** message:
```json
{
  "match_id": "the_match_id",
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
Player login happens by **dvc** via **login-player** message:
```json
{
  "match_id": "the_match_id",
  "user_id": "the_user_id",
  "team_id": "the_team_id"
}
```