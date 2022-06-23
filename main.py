import os
from datetime import timedelta

import requests
import requests_cache
from flask import abort, Flask, Response
from ics import Calendar, Event

from util import ElphiEvent

ICS_CREATOR = os.getenv("ICS_CREATOR", "elphi-calendar")

app = Flask(__name__)
session = requests_cache.CachedSession(
    'events',
    expire_after=timedelta(days=1),
    backend='memory'
)


def fetch_event(event_id):
    url = f"https://www.elbphilharmonie.de/de/api/booking/evis/{event_id}/"
    response = session.get(url)
    if response.status_code != 200:
        return None
    event = ElphiEvent(response.json())
    response = session.get(event.ics_url)
    response.encoding = "utf-8"
    calendar = Calendar(response.text)
    event.calendar.events.update(calendar.events)
    return event


@app.route(f"/merkliste/<user>")
def merkliste(user: str):
    response = requests.get(f"https://merkliste.elbphilharmonie.de/api/{user}")
    if response.status_code == 404:
        abort(404)
    favorites = response.json()
    events = []
    for event_id in favorites["events"]:
        event = fetch_event(event_id)
        if event:
            events.append(event)

    calendar = Calendar(creator=ICS_CREATOR)
    for event in events:
        ics_event: Event
        for ics_event in event.calendar.events:
            ics_event.name = event.subtitle_de
            ics_event.description = event.title_de + "\n\n" + ics_event.description
        calendar.events.update(event.calendar.events)
    return Response(calendar, mimetype="text/calendar")


@app.route("/health")
def health():
    return "OK"
