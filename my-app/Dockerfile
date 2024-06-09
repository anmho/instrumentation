FROM --platform=linux/amd64 golang:1.22.2 AS build

WORKDIR /app
# Allow caching of dependencies and prevent reinstalls
COPY go.mod ./
COPY go.sum ./

RUN go mod download && go mod verify

COPY main.go ./

RUN go build -o /my-app

FROM --platform=linux/amd64 ubuntu

COPY --from=build /my-app /my-app

ENTRYPOINT ["/my-app"]