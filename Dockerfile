FROM golang:1.10 as builder

WORKDIR ${GOPATH}/src/github.com/dispatchframework/dispatch-events-eventgrid

COPY ["driver.go", "Gopkg.lock", "Gopkg.toml", "./"]

RUN go get -u github.com/golang/dep/cmd/dep
RUN dep ensure

RUN CGO_ENABLED=0 GOOS=linux go build -a -o /dispatch-events-eventgrid


FROM scratch

COPY --from=builder /dispatch-events-eventgrid /

ENTRYPOINT [ "/dispatch-events-eventgrid" ]