FROM node:16.13.0-slim as build-step

# Copy the current directory contents into the container at /app
ADD ./frontend /app/frontend

# Set the working directory to /app
WORKDIR /app/frontend
RUN npm install --silent
RUN npm run build --silent

FROM python:3.9-slim

# Need to install a C compiler for uwsgi
RUN apt-get update && apt-get install -y gcc

ADD ./server /app/server

COPY --from=build-step /app/frontend/build /app/server/flaskApp/static

WORKDIR /app/server

RUN pip3 install -r requirements.txt

RUN mkdir /data

ENV IS_LIVE=true

CMD ["uwsgi", "--socket", "0.0.0.0:5002", "--protocol=http", "--enable-threads", "-w", "wsgi:app"]