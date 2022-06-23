import os
from datetime import timedelta

import requests
import requests_cache
from dateutil.parser import isoparse
from flask import abort, Flask, Response
from ics import Calendar, Event

from util import clean_description, clean_location

ICS_CREATOR = os.getenv("ICS_CREATOR", "elphi-calendar")

app = Flask(__name__)
session = requests_cache.CachedSession(
    'events',
    expire_after=timedelta(days=1)
)


@app.route(f"/<user>")
def merkliste(user: str):
    response = requests.get(f"https://merkliste.elbphilharmonie.de/api/{user}")
    if response.status_code == 404:
        abort(404)
    favorites = response.json()
    events = []
    for event_id in favorites["events"].keys():
        response = session.get(
            f"https://www.elbphilharmonie.de/de/api/booking/evis/{event_id}/"
        )
        if response.status_code == 200:
            events.append(response.json())

    calendar = Calendar(creator=ICS_CREATOR)
    for event in events:
        start_date = isoparse(event["date_start"])
        end_date = isoparse(event["date_end"]) \
            if event.get("date_end") \
            else (start_date + timedelta(hours=2))
        calendar.events.add(Event(
            name=event["title_de"],
            location=clean_location(event["room_dispname"]),
            begin=start_date,
            end=end_date,
            url=event["website_url"],
            description=event["subtitle_de"] + "\n\n" + clean_description(
                event["description_long_de"])
        ))
    return Response(calendar, mimetype="text/calendar")
