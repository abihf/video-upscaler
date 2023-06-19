# GO builder
FROM golang:1.20-alpine AS app-build
WORKDIR /workspace

ADD go.mod go.sum ./
RUN go mod download

ADD . ./
RUN --mount=type=cache,target=/root/.cache/go-build CGO_ENABLED=0 GOOS=linux go build -a -o video-upscaler .
# RUN CGO_ENABLED=0 GOOS=linux go build -a -o scanner ./cli/scanner

FROM alpine AS tini
ARG TINI_VERSION=v0.19.0
ADD https://github.com/krallin/tini/releases/download/${TINI_VERSION}/tini /tini
RUN chmod +x /tini

# ========================================================= #


FROM ghcr.io/stargz-containers/ubuntu:22.04-esgz

FROM abihf/video-upscaler:base AS worker
# COPY --from=tini /tini /tini
# ENTRYPOINT ["/tini", "--"]
COPY script.vpy models /upscale/
COPY --link --from=app-build /workspace/video-upscaler /usr/local/bin/video-upscaler
CMD [ "/usr/local/bin/video-upscaler", "worker" ]
