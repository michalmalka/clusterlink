FROM alpine:3.14

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY  ./bin/manager .
USER 65532:65532

ENTRYPOINT ["/manager"]
