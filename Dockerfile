FROM smartentry/debian:bullseye

MAINTAINER Yifan Gao <docker@yfgao.com>

ADD .docker /opt/liims/.docker/

ENV ASSETS_DIR=/opt/liims/.docker

RUN smartentry.sh build

ADD . /opt/liims/

VOLUME /srv
