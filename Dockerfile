FROM golang:1.15 AS build
ENV CGO_ENABLED=0
RUN go get -v github.com/go-delve/delve/cmd/dlv
WORKDIR /go/src/github.com/inloco/artifactcache
COPY ./go.mod ./go.mod
COPY ./go.sum ./go.sum
RUN go mod download
COPY ./*.go ./
RUN go install -a -gcflags 'all=-N -l' -ldflags '-d -extldflags "-fno-PIC -static"' -tags 'netgo osusergo static_build' -trimpath -v ./...

FROM gcr.io/distroless/static:nonroot AS runtime
COPY --from=build /go/bin/dlv /sbin/dlv
COPY --from=build /go/bin/artifactcache /sbin/init
ENTRYPOINT ["/sbin/init"]
