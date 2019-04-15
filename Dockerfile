FROM daocloud.io/library/debian:latest
MAINTAINER electricface <songwentai@linuxdeepin.com>
ADD bin /app/bin
RUN ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
	echo "Asia/Shanghai" > /etc/timezone