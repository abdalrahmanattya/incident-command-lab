FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/incidentlab ./cmd/gateway && CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/payment ./cmd/payment && CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/notification ./cmd/notification
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/incidentlab /incidentlab
COPY --from=build /out/payment /payment
COPY --from=build /out/notification /notification
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/incidentlab"]
