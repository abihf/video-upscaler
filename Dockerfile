# GO builder
FROM golang:1.21rc2-alpine AS source
WORKDIR /workspace

ADD go.mod go.sum ./
RUN go mod download

ADD . ./

FROM source AS worker-build
RUN --mount=type=cache,target=/root/.cache/go-build CGO_ENABLED=0 GOOS=linux go build -a -o video-upscaler .


# ========================================================= #

FROM scratch AS cli
COPY --link --from=worker-build /workspace/video-upscaler /usr/bin/video-upscaler
ENTRYPOINT [ "/usr/bin/video-upscaler" ]

FROM golang:1.20-alpine AS tini
ARG TINI_VERSION=v0.19.0
ADD https://github.com/krallin/tini/releases/download/${TINI_VERSION}/tini /tini
RUN chmod +x /tini

FROM ghcr.io/abihf/video-upscaler:base AS worker
COPY --from=tini /tini /tini
ENTRYPOINT ["/tini", "--"]
COPY script.py /upscale/
COPY --link --from=worker-build /workspace/video-upscaler /usr/bin/video-upscaler
CMD [ "/usr/bin/video-upscaler", "worker" ]

