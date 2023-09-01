# Use this Dockerfile if you want to "go build" on docker (if you don't have go installed)

### BUILD IMAGE ###

FROM golang:1.21-alpine AS compile-stage

# git needed by go get / go build
RUN apk add git

# add a non-root user for running the application
RUN addgroup -g 1000 app
RUN adduser \
    -D \
    -g "" \
    -h /app \
    -G app \
    -u 1000 \
    app
WORKDIR /app

# create volume directory
RUN mkdir data
# install dependencies before copying everything else to allow for caching
COPY go.mod go.sum ./
RUN go get -d ./...
# copy the code into the build image
COPY . .

# set permissions for the app user
RUN chown -R app /app
RUN chmod -R +rwx /app

# build the application
ARG CGO_ENABLED=0
ARG GOOS=linux
RUN go build -a -installsuffix cgo -o beacon .

### RUNTIME IMAGE ###

FROM scratch as runtime-stage
# copy the user files and switch to app user
COPY --from=compile-stage /etc/passwd /etc/passwd
COPY --from=compile-stage /etc/group /etc/group
COPY --from=compile-stage /etc/shadow /etc/shadow
USER app
# copy the binary and static files from the build image
COPY --chown=app:app --from=compile-stage /app/beacon /beacon
COPY --chown=app:app --from=compile-stage /app/static /static
# copy the data folder with the correct permissions for the volume mount
COPY --chown=app:app --from=compile-stage /app/data /data
ENTRYPOINT ["/beacon"]
