FROM smartentry/debian:bullseye

MAINTAINER Yifan Gao <docker@yfgao.com>

ADD . /opt/liims

ENV ASSETS_DIR=/opt/liims/.docker

RUN smartentry.sh build

VOLUME /srv
