FROM golang:alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/tilapou ./cmd/tilapou

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /out/tilapou /tilapou

USER nonroot:nonroot
EXPOSE 8080

ENTRYPOINT ["/tilapou", "serve"]
