FROM golang:1-alpine

COPY *.go src/github.com/vadd/verbum/
COPY index.gohtml index.gohtml
RUN go install github.com/vadd/verbum

EXPOSE 8080 10443

CMD ["verbum"]
