FROM debian:11

ARG APT_SOURCE=https://mirrors.ustc.edu.cn
ENV APT_SOURCE=$APT_SOURCE

RUN sed -Ei "s,https?://(deb|security)\.debian\.org,$APT_SOURCE,g" /etc/apt/sources.list && \
    apt-get update && \
    apt-get -y upgrade && \
    apt-get install --no-install-recommends --yes \
        debootstrap build-essential libcurl4-openssl-dev libx11-dev libxext-dev libxss-dev \
        curl ca-certificates squashfs-tools rsync && \
    apt-get clean

WORKDIR /opt/liims
ADD . /opt/liims/
VOLUME /srv/dest
CMD ["./docker-run.sh"]
