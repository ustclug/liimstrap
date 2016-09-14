FROM smartentry/alpine:3.4-0.3.5

MAINTAINER Yifan Gao <docker@yfgao.com>

ADD . /opt/monitor

ENV ASSETS_DIR=/opt/monitor/.docker

RUN smartentry.sh build
