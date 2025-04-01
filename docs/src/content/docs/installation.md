---
title: Installation
description: How to install Railpack
---

Railpack is available as a CLI tool. The latest release is available [on
GitHub](https://github.com/salamer/railpack/releases).

The BuildKit frontend is available as a [Docker image on
GHCR](https://github.com/salamer/railpack/pkgs/container/railpack-frontend).

## Curl

Download Railpack from GH releases and install automatically

```sh
curl -sSL https://railpack.com/install.sh | sh
```

## GitHub Releases

Go to the [latest release](https://github.com/salamer/railpack/releases) and
download the `railpack` binary for your platform.

## From Source

```sh
git clone https://github.com/railwayapp/railpack.git
cd railpack
go build -o railpack ./cmd/...

./railpack --help
```
