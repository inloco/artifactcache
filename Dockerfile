FROM golang:1.15 AS build
WORKDIR /go/src/github.com/inloco/artifactcache
COPY ./go.mod ./go.mod
COPY ./go.sum ./go.sum
RUN go mod download
COPY ./*.go ./
RUN CGO_ENABLED=0 go install -a -ldflags '-d -extldflags "-fno-PIC -static" -s -w' -tags 'netgo osusergo static_build' -trimpath -v ./...

FROM gcr.io/distroless/static:nonroot AS runtime
COPY --from=build /go/bin/artifactcache /sbin/init
ENTRYPOINT ["/sbin/init"]
