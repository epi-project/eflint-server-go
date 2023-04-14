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

### Running the server
After installing, the server can be started by running the following command:
```
./eflint-server
```

### Interacting with the server
To run eFLINT programs, you can use the eFLINT to JSON converter (TBD).
