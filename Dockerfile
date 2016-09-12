FROM smartentry/archlinux:0.3.2

MAINTAINER Yifan Gao <docker@yfgao.com>

ADD . /opt/liims

ENV ASSETS_DIR=/opt/liims/docker

RUN smartentry.sh build

VOLUME /srv

WORKDIR /opt/liims

CMD ["/usr/bin/run.sh"]
