FROM smartentry/alpine:3.4-0.3.4

MAINTAINER Yifan Gao <docker@yfgao.com>

ADD .docker $ASSETS_DIR

ADD . /var/www

RUN smartentry.sh build
