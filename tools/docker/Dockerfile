
FROM alpine:3.15

EXPOSE 162
EXPOSE 80

# Package inventory:
#     https://pkgs.alpinelinux.org/packages?page=2&branch=edge
RUN apk add --no-cache --upgrade bash && apk add net-snmp-tools

RUN mkdir -p /opt/trapex/bin /opt/trapex/etc /opt/trapex/log
COPY trapex /opt/trapex/bin
COPY tools/trapex.conf /opt/trapex/etc

# Install root bash history
COPY tools/docker/root_bash_history /root/.bash_history
COPY tools/docker/root_vimrc /root/.vimrc

CMD ["/opt/trapex/bin/trapex", "-c", "/opt/trapex/etc/trapex.conf"]
