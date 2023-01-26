ARG REGISTRY
ARG IMAGE_PREFIX

FROM  ${REGISTRY}/${IMAGE_PREFIX}golang:1.16-alpine AS build

WORKDIR /code

COPY go.mod go.sum /code/
RUN go mod download
COPY . .

RUN go build -o bin/datagen cmd/datagen/main.go
RUN go build -o bin/dynamo_table cmd/dynamo_table/main.go

FROM ${REGISTRY}/${IMAGE_PREFIX}alpine:3.12.2 AS production

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /code/bin/* /bin/

ENTRYPOINT ["datagen"]
