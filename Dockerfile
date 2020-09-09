FROM ubuntu:bionic
RUN apt-get update
RUN apt-get -y install rsyslog ca-certificates
env AWS_ACCESS_KEY ""
env AWS_SECRET_ACCESS_KEY ""
env AWS_SESSION_TOKEN ""
COPY ./50-transport.conf /etc/rsyslog.d/50-transport.conf
COPY ./log-transporter /usr/local/bin/log-transporter
RUN mkdir /app
COPY ./test.sh /app/test.sh