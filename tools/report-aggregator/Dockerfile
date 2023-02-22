ARG REGISTRY

FROM ${REGISTRY}/golang:1.20-alpine AS builder

WORKDIR /code/
COPY go.mod go.sum /code/
RUN go mod download
COPY . /code

RUN go build -o report-aggregator .

FROM ${REGISTRY}/alpine:3.12.2 AS production

COPY --from=builder /code/report-aggregator /app/report-aggregator
ENV PATH="/app/:${PATH}"

ENTRYPOINT ["/app/report-aggregator"]
