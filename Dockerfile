ARG BASEIMG="alpine:3.15"
ARG BUILDIMG="golang:1.18-alpine3.15"
FROM $BUILDIMG as builder

ARG APP_NAME="rss_checker"
ENV GOPATH=""

RUN apk --no-cache add git

COPY . /go/

RUN cd /go \
	&& go build -o /${APP_NAME}

FROM $BASEIMG
LABEL maintainer="Nate Catelli <ncatelli@packetfire.org>"
LABEL description="Container for rss_checker"

ARG SERVICE_USER="service"
ARG APP_NAME="rss_checker"

RUN addgroup ${SERVICE_USER} \
    && adduser -D -G ${SERVICE_USER} ${SERVICE_USER}

COPY --from=builder /${APP_NAME} /opt/${APP_NAME}/bin/${APP_NAME}

RUN chown -R ${SERVICE_USER}:${SERVICE_USER} /opt/${APP_NAME}/bin/${APP_NAME} \
    && chmod +x /opt/${APP_NAME}/bin/${APP_NAME}

WORKDIR "/opt/${APP_NAME}/"
USER ${SERVICE_USER}

ENTRYPOINT [ "/opt/rss_checker/bin/rss_checker" ]
CMD [ "-h" ]
