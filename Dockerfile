FROM golang:1-alpine

RUN set -ex \
  && apk add --no-cache --virtual .build-deps curl \
  && apk add --no-cache --virtual .build-deps-git git \
  && curl https://glide.sh/get | sh \
  && apk del .build-deps

COPY glide.* src/github.com/vadd/verbum/
RUN set -ex \
  && (cd src/github.com/vadd/verbum/ && glide install)
RUN apk del .build-deps-git

COPY *.go src/github.com/vadd/verbum/
COPY templates templates
RUN go install github.com/vadd/verbum

EXPOSE 8080
CMD ["verbum"]
