FROM golang:1.14-alpine AS build

#setting SP credentials
RUN apk add --no-cache git

# Set the Current Working Directory inside the container
WORKDIR $GOPATH/src/tlsretriever

# We want to populate the module cache based on the go.{mod,sum} files.
COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

# Unit tests
RUN CGO_ENABLED=0 go test -v

# Build the Go app
RUN go build -o /tlsretriever

# Start fresh from a smaller image
FROM alpine:3.9 

RUN apk add ca-certificates

COPY --from=build /tlsretriever /tlsretriever


# Run the binary program produced by `go install`
ENTRYPOINT /tlsretriever
