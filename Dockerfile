FROM golang:1.11-alpine AS build
ENV AZURE_CLIENT_ID="7ec68311-0abc-4569-9d10-bea623af6e51" AZURE_CLIENT_SECRET="ba35d05c-60b4-4c37-9490-4a90306a7ad4" AZURE_TENANT_ID="b8d7ad48-53f4-4c29-a71c-0717f0d3a5d0"
WORKDIR /src/
COPY main.go go.* /src/
RUN CGO_ENABLED=0 go build -o /bin/demo
FROM scratch COPY --from=build /bin/demo /bin/demo
ENTRYPOINT ["/bin/demo"] 
