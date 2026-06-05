# Build the calendar backend as a static binary, then run it on a minimal distroless image.
FROM golang:1.26 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /calendar-server ./cmd/server

FROM gcr.io/distroless/static-debian12
COPY --from=build /calendar-server /calendar-server
EXPOSE 9090
ENTRYPOINT ["/calendar-server"]
