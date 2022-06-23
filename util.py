from html2text import html2text


def clean_location(location: str) -> str:
    return location.replace("<strong>", "").replace("</strong>", ":")


def clean_description(desc: str) -> str:
    return html2text(desc, bodywidth=0).strip()
