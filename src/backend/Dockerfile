FROM golang:latest AS build

# ARG PB_VERSION=0.22.23

WORKDIR /src
# download and unzip PocketBase
# ADD https://github.com/pocketbase/pocketbase/releases/download/v${PB_VERSION}/pocketbase_${PB_VERSION}_linux_amd64.zip /tmp/pb.zip
# RUN unzip /tmp/pb.zip -d /pb/

# uncomment to copy the local pb_migrations dir into the image
# COPY ./pb_migrations /pb/pb_migrations

# uncomment to copy the local pb_hooks dir into the image
# COPY ./pb_hooks /pb/pb_hooks

COPY src/main.go /src/main.go
RUN go mod init pocketbase-scriptflow && go mod tidy
RUN CGO_ENABLED=0 go build

FROM alpine:latest AS server

RUN apk add --no-cache \
    unzip \
    ca-certificates

# copy executable from build
COPY --from=build /src/pocketbase-scriptflow /pb/pocketbase-scriptflow

EXPOSE 8080

# start PocketBase
CMD ["/pb/pocketbase-scriptflow", "serve", "--http=0.0.0.0:8080"]
