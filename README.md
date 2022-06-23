# Elphi Calendar

![Docker Image](https://github.com/codello/elphi-calendar/actions/workflows/build.yml/badge.svg)
![MIT License](https://img.shields.io/github/license/codello/elphi-calendar)

This simple python server serves an ICS version of your favorites for the [Elbphilharmonie](http://elbphilharmonie.de).

## Quick Start

The quickest way to run the calendar API is to use the docker container:

```shell
docker run -p 8080:8080 ghcr.io/codello/elphi-calendar
```

This will run the app on port 8080. You can now access your calendar at

```
http://localhost:8080/merkliste/<user-id>
```

See the next paragraph on how to find your user ID.

### The User ID

The Elbphilharmonie favorite lists of every user is accessible without authentication. All you need is the user’s ID. To find your ID, you can do the following:

- Open your web browser, go to [Your Account](https://shop.elbphilharmonie.de/de/meine-daten/).

- Open the JavaScript Console and enter `elbphilharmonie.ActivityId`. This will display an activity id in your console. Copy the value (whithout quotes).

- Run the following command that will print your account ID:
  ```shell
  ACTIVITY_ID=<your activity ID>
  curl -sS 'https://shop-services.elbphilharmonie.de/Activities/validateactivityid.json?ActivityId=$ACTIVITY_ID' | jq -r .User.UserId
  ```

## Caching

The `elphi-calendar` will cache event details (but not your favorites list) for one day. This dramatically reduces the number of API calls and increases update speed. The disadvantage is that changes to events will only become visible after up to a day.