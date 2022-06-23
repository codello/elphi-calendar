FROM python:alpine

ENV PYTHONDONTWRITEBYTECODE 1
ENV PYTHONUNBUFFERED 1

WORKDIR /app
RUN pip install --upgrade pip waitress
COPY requirements.txt /app
RUN pip install -r requirements.txt

COPY *.py /app

CMD ["waitress-serve", "main:app"]