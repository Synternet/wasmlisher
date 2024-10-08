#
# 1. Build Container
#
FROM golang:1.21 AS build
ARG VERSION CI_JOB_TOKEN CI_SERVER_HOST NATS_HOSTS_PUB NATS_HOSTS_SUB NATS_CREDS NATS_JWT NATS_NKEY NATS_ACC_NKEY

RUN bash -c 'echo -e "machine gitlab.com\nlogin gitlab-ci-token\npassword ${CI_JOB_TOKEN}" > ~/.netrc'
RUN git config --global url.https://gitlab-ci-token:${CI_JOB_TOKEN}@${CI_SERVER_HOST}.insteadOf https://${CI_SERVER_HOST}

COPY . /home/src
WORKDIR /home/src

# Build components.
RUN CGO_ENABLED=1 go build -v -o . -installsuffix cgo -ldflags="-w -s" ./...

#
# 2. Runtime Container
#
FROM golang:1.21

WORKDIR /home/app
COPY --from=build /home/src/wasmlisher .

CMD ["sh", "-c", "\
./wasmlisher \
--nats-acc-nkey ${NATS_ACC_NKEY} \
--nats-pub-url ${NATS_HOSTS_PUB} \
--nats-sub-url ${NATS_HOSTS_SUB} \
--nats-pub-jwt ${NATS_PUB_JWT} \
--nats-pub-nkey ${NATS_PUB_NKEY} \
--nats-sub-jwt ${NATS_SUB_JWT} \
--nats-sub-nkey ${NATS_SUB_NKEY} \
--name ${NAME} \
--config ${CONFIG} \
--cfInterval ${CONFIG_INTERVAL} \
start \
"]