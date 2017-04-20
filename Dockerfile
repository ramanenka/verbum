FROM golang:1-alpine

RUN set -ex \
  && apk add --no-cache git

RUN go get golang.org/x/text/unicode/norm
RUN go get golang.org/x/text/transform
RUN go get github.com/gorilla/mux
COPY *.go src/github.com/verbumby/verbum/
COPY index.gohtml index.gohtml
RUN go install github.com/verbumby/verbum

COPY statics/node_modules/components-font-awesome/css/font-awesome.min.css statics/node_modules/components-font-awesome/css/font-awesome.min.css
COPY statics/node_modules/components-font-awesome/fonts/* statics/node_modules/components-font-awesome/fonts/

EXPOSE 8080 10443

CMD ["verbum"]
