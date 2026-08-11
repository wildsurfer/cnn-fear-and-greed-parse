FROM golang:1-alpine AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /cnnfag ./cmd/cnnfag

FROM scratch
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /cnnfag /cnnfag
LABEL io.modelcontextprotocol.server.name="io.github.wildsurfer/cnnfag"
LABEL org.opencontainers.image.source="https://github.com/wildsurfer/cnn-fear-and-greed-parse"
ENTRYPOINT ["/cnnfag", "mcp"]
