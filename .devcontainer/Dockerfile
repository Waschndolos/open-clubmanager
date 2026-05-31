FROM mcr.microsoft.com/devcontainers/base:ubuntu-24.04

ARG NODE_MAJOR=22
ARG GO_VERSION=1.24.3
ARG RUST_VERSION=1.77.2

ENV DEBIAN_FRONTEND=noninteractive \
    CARGO_HOME=/usr/local/cargo \
    RUSTUP_HOME=/usr/local/rustup \
    PATH=/usr/local/go/bin:/usr/local/cargo/bin:${PATH}

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        bash-completion \
        build-essential \
        ca-certificates \
        curl \
        file \
        git \
        gnupg \
        jq \
        libasound2-dev \
        libayatana-appindicator3-dev \
        libglib2.0-dev \
        librsvg2-dev \
        libsqlite3-dev \
        libsoup-3.0-dev \
        libssl-dev \
        libwebkit2gtk-4.1-dev \
        libxdo-dev \
        libgtk-3-dev \
        default-mysql-client \
        patchelf \
        pkg-config \
        python3 \
        python3-pip \
        python3-venv \
        sqlite3 \
        unzip \
        xz-utils \
        zip \
    && rm -rf /var/lib/apt/lists/*

RUN ARCH="$(dpkg --print-architecture)" \
    && case "${ARCH}" in \
        amd64) NODE_ARCH='x64' ;; \
        arm64) NODE_ARCH='arm64' ;; \
        *) echo "Unsupported architecture: ${ARCH}" >&2; exit 1 ;; \
    esac \
    && NODE_VERSION="$(curl -fsSL "https://nodejs.org/dist/latest-v${NODE_MAJOR}.x/SHASUMS256.txt" | sed -n '1s|.*node-v\([0-9.]*\)-.*|\1|p')" \
    && NODE_DIST="node-v${NODE_VERSION}-linux-${NODE_ARCH}" \
    && curl -fsSL "https://nodejs.org/dist/v${NODE_VERSION}/${NODE_DIST}.tar.xz" -o /tmp/node.tar.xz \
    && curl -fsSL "https://nodejs.org/dist/v${NODE_VERSION}/SHASUMS256.txt" -o /tmp/node-shasums.txt \
    && NODE_TAR_SHA256="$(grep " ${NODE_DIST}.tar.xz$" /tmp/node-shasums.txt | awk '{print $1}')" \
    && [ -n "${NODE_TAR_SHA256}" ] \
    && echo "${NODE_TAR_SHA256}  /tmp/node.tar.xz" | sha256sum -c - \
    && tar -xJf /tmp/node.tar.xz -C /usr/local --strip-components=1 --no-same-owner \
    && rm -f /tmp/node.tar.xz /tmp/node-shasums.txt \
    && corepack enable

RUN ARCH="$(dpkg --print-architecture)" \
    && case "${ARCH}" in \
        amd64) GO_ARCH='amd64' ;; \
        arm64) GO_ARCH='arm64' ;; \
        *) echo "Unsupported architecture: ${ARCH}" >&2; exit 1 ;; \
    esac \
    && curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-${GO_ARCH}.tar.gz" -o /tmp/go.tar.gz \
    && rm -rf /usr/local/go \
    && tar -C /usr/local -xzf /tmp/go.tar.gz \
    && rm /tmp/go.tar.gz

RUN mkdir -p "${CARGO_HOME}" "${RUSTUP_HOME}" \
    && curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs -o /tmp/rustup-init.sh \
    && sh /tmp/rustup-init.sh -y --profile minimal --default-toolchain "${RUST_VERSION}" \
    && rustup component add clippy rustfmt \
    && rm /tmp/rustup-init.sh \
    && chown -R vscode:vscode "${CARGO_HOME}" "${RUSTUP_HOME}"

RUN npm --version \
    && node --version \
    && go version \
    && cargo --version \
    && rustc --version

