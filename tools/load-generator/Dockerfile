ARG REGISTRY
ARG IMAGE_PREFIX
FROM ${REGISTRY}/${IMAGE_PREFIX}golang:1.20-alpine AS builder

WORKDIR /code/
COPY go.mod go.sum /code/
RUN go mod download
COPY . /code

RUN go build -o load-generator .

FROM ${REGISTRY}/${IMAGE_PREFIX}alpine:3.12.2 AS production

COPY --from=builder /code/load-generator /app/load-generator
ENV PATH="/app/:${PATH}"

ENTRYPOINT ["/app/load-generator"]
