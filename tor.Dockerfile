FROM alpine:latest
RUN apk add --no-cache tor netcat-openbsd
COPY torrc /etc/tor/torrc
RUN mkdir -p /var/lib/tor && chown -R tor:tor /var/lib/tor /etc/tor
USER tor
EXPOSE 9050
CMD ["tor", "-f", "/etc/tor/torrc"]
