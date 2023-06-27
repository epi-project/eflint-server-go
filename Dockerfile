# DOCKERFILE for the eFLINT server
# 
# Dockerfile for the eFLINT Server. Can be used to install and launch it without issues.
# 


##### BUILD #####
# Open the GO image to build the server
FROM golang:1.18-alpine AS build

# Copy the files
RUN mkdir -p /build/bin
COPY . /build
WORKDIR /build

# Install
RUN go build -o ./bin ./...



##### RUN #####
FROM alpine

# Install runtime dependencies

# Copy the binary
COPY --from=build /build/bin/eflint-server /eflint-server
RUN chmod +x /eflint-server

# Set it as entrypoint
ENTRYPOINT [ "/eflint-server" ]
