from dataclasses import dataclass

from ics import Calendar


class ElphiEvent:
    __slots__ = ["data", "calendar"]

    def __init__(self, data: dict):
        self.data = data
        self.calendar: Calendar = Calendar()

    @property
    def title_de(self) -> str:
        return self.data["title_de"]

    @property
    def subtitle_de(self) -> str:
        return self.data["subtitle_de"]

    @property
    def website_url(self) -> str:
        return self.data["website_url"]

    @property
    def ics_url(self) -> str:
        return f"{self.website_url}.ics"
