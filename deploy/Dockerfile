FROM new.cicd-jfrog.telkomsel.co.id/docker/alpine:3.19

ARG ARCH

ENV APP_PORT=443
ENV SERVER_LISTEN_TIMEOUT=10

# hadolint ignore=DL3018
RUN \
    apk --no-cache add curl && \
    apk --no-cache add inetutils-telnet && \
    apk --no-cache add tzdata

WORKDIR /app

COPY bin/main-$ARCH /app/main
RUN chmod +x /app/main
CMD ["/app/main"]
