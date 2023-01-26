ARG REGISTRY
FROM ${REGISTRY}/python:3.9-slim

RUN apt-get update && apt-get install -y curl libc6-dev libssl-dev autoconf automake libtool g++ git make libz-dev
RUN curl -L -o /usr/local/bin/kubectl https://dl.k8s.io/release/v1.20.4/bin/linux/$(case $(uname -m) in x86_64) echo x86_64;; aarch64) echo arm64;; esac)/kubectl && chmod +x /usr/local/bin/kubectl
RUN bash -c 'set -o pipefail && curl https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3 | bash'
RUN pip install poetry

WORKDIR /app
RUN git clone https://github.com/aerospike/aerospike-client-c \
    && cd /app/aerospike-client-c \
    && git checkout 5.1.0 \
    && git submodule update --init

COPY tools/e2e/gcc /usr/local/bin/gcc
COPY tools/e2e/gcc /usr/local/bin/cc

RUN cd /app/aerospike-client-c \
    && make

# Symlink the target to the (wrong on ARM64) path expected by the Python client.
RUN cd /app/aerospike-client-c/target \
    && [ -d Linux-x86_64 ] || ln -s Linux-aarch64 Linux-x86_64

RUN mkdir -p /app/tools/e2e /app/infrastructure/charts
WORKDIR /app/tools/e2e
COPY tools/e2e/pyproject.toml tools/e2e/poetry.lock /app/tools/e2e/
RUN DOWNLOAD_C_CLIENT=0 AEROSPIKE=/app/aerospike-client-c AEROSPIKE_C_HOME=/app/aerospike-client-c poetry install

COPY tools/e2e/setup.cfg /app/tools/e2e/
COPY deployment/infrastructure/charts/bidder/ /app/infrastructure/charts/bidder
COPY tools/e2e/tests/ /app/tools/e2e/tests
COPY tools/e2e/entrypoint.sh /app/tools/e2e/entrypoint.sh
ENTRYPOINT ["/app/tools/e2e/entrypoint.sh"]
