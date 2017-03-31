FROM golang:1-alpine

RUN set -ex \
  && apk add --no-cache --virtual .build-deps curl \
  && apk add --no-cache --virtual .build-deps-git git \
  && curl https://glide.sh/get | sh \
  && apk del .build-deps

RUN apk del .build-deps-git

COPY *.go src/github.com/vadd/verbum/
COPY index.gohtml index.gohtml
RUN go install github.com/vadd/verbum

EXPOSE 8080
CMD ["verbum"]
