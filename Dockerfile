FROM golang:1.14-alpine AS build

#setting SP credentials
ENV AZURE_CLIENT_ID="7ec68311-0abc-4569-9d10-bea623af6e51" AZURE_CLIENT_SECRET="ba35d05c-60b4-4c37-9490-4a90306a7ad4" AZURE_TENANT_ID="b8d7ad48-53f4-4c29-a71c-0717f0d3a5d0"

RUN apk add --no-cache git

# Set the Current Working Directory inside the container
WORKDIR $GOPATH/src/certval

# We want to populate the module cache based on the go.{mod,sum} files.
COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

# Unit tests
RUN CGO_ENABLED=0 go test -v

# Build the Go app
RUN go build -o /app/certval

# Start fresh from a smaller image
FROM alpine:3.9 
RUN apk add ca-certificates

COPY --from=build /app/certval /app/certval


# Run the binary program produced by `go install`
ENTRYPOINT /app/certval
