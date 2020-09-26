# MASC - Flows

Mission Airsoft Control

This paper describes common flows in Mission Airsoft Control, developed by _TODO_ in Germany.
All messages include the following field ```"meta"``` with the message type in lower case:
```json
{
  "type": "the_message_type",
  "id": "the_sender_id"
}
```
If an errors occurs during handling messages, or the message contains invalid data, an **ERROR** message with the specific error code:
```json
{
  "error_code": 0,
  "message": "the_error_message"
}
``` 
If not stated otherwise, **dvc** references the client.

## Gatekeeping
### Login

This describes the login process for all devices.

Referring:

- **nd** as the device which wants to log in
- **srv** as the server

Flow:

- **nd** sends a **HELLO**-Message:

  ```json
  {
    "name": "the_device_name",
    "roles": []
  }
  ```
- The **srv** responds with a **WELCOME**-Message which also contains the assigned id in the ```meta``` field:
```json
{
  "server_name": "the_server_name"
}
```

## Scheduling
Flows for scheduling games and retrieving schedules.
### Retrieve scheduling
Unfinished games get retrieved by the **dvc** via **GET_SCHEDULE**:
```json
{}
```
The **srv** responds via **SCHEDULE**:
```json
{
  "schedule": [
    {
      "id": "the_event_id",
      "type": "game",
      "title": "the_title",
      "description": "the_description",
      "start_time": "the_events_start_time",
      "end_time": "the_events_end_time"
    },
    {}
  ]
}
```

