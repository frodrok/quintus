FROM golang:1.23-alpine AS build
WORKDIR /src
COPY . .
RUN go build -o /mock-proxy .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /mock-proxy /mock-proxy
EXPOSE 8888
ENTRYPOINT ["/mock-proxy"]