# Scalable implementation of the eFLINT language

## Table of Contents
* [Introduction](#introduction)
* [Getting Started](#getting-started)

## Introduction
This repository contains a Go implementation of the eFLINT language. It offers a
server that can be used to run eFLINT programs. The server is scalable and can
be used to run multiple programs at the same time. To interact with it,
JSON messages can be sent to the server. The server will respond with a JSON
message containing the result of the program. The exact specification of the
JSON messages can be found in the [JSON specification](https://gitlab.com/eflint/json-specification).

## Getting Started

### Prerequisites
Requires Go to be set up on the target device. For more information, see the
[Go documentation](https://golang.org/doc/install).

### Installing
TBD

#### Docker
You can also build a Docker container with the eFLINT server directly. To do so, be sure that [Docker](https://docs.docker.com/engine/install/) is installed, and then run:
```bash
docker build --tag eflint-server:latest -f Dockerfile .
```
in the root of the repository.

> If you are using the [`Buildx`]-plugin as default, don't forget to specify you want to load the result:
> ```bash
> docker build --load --tag eflint-server:latest -f Dockerfile .
> ```

### Running the server
After installing, the server can be started by running the following command:
```
./eflint-server
```

#### Docker
To run the built Docker container, simply run the following command:
```bash
docker run -it --rm -p 8080:8080 eflint-server
```
to run it in your terminal, or
```bash
docker run --name eflint-server -d -p 8080:8080 eflint-server
```
to run it in the background.

### Interacting with the server
To run eFLINT programs, you can use the eFLINT to JSON converter (TBD).
