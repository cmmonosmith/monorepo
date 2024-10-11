FROM ubuntu:latest

ARG APP_NAME

#ENV APP_NAME="${APP_NAME}"

COPY bin/${APP_NAME} /usr/local/bin/

# Set the entrypoint to run the executable when the container starts
ENTRYPOINT ["/usr/local/bin/${APP_NAME}"]