FROM golang:1-alpine

RUN set -ex \
  && apk add --no-cache git

RUN go get golang.org/x/text/unicode/norm
RUN go get golang.org/x/text/transform
COPY *.go src/github.com/vadd/verbum/
COPY index.gohtml index.gohtml
RUN go install github.com/vadd/verbum

EXPOSE 8080 10443

CMD ["verbum"]
