FROM debian:12

ARG APT_SOURCE=http://mirrors.ustc.edu.cn
ENV APT_SOURCE=$APT_SOURCE

RUN sed -Ei "s,https?://(deb|security)\.debian\.org,$APT_SOURCE,g" /etc/apt/sources.list.d/debian.sources && \
    apt-get update && \
    apt-get -y upgrade && \
    apt-get install --no-install-recommends --yes \
        gcc libc6-dev libx11-dev libxss-dev \
        curl ca-certificates debootstrap rsync squashfs-tools && \
    apt-get clean

WORKDIR /opt/liims
ADD . /opt/liims/
VOLUME /srv/dest
CMD ["./docker-run.sh"]
