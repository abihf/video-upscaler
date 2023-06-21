# GO builder
FROM golang:1.20-alpine AS source
WORKDIR /workspace

ADD go.mod go.sum ./
RUN go mod download

ADD . ./

FROM source AS cli-build
RUN --mount=type=cache,target=/root/.cache/go-build CGO_ENABLED=0 GOOS=linux go build -tags noworker -a -o video-upscaler .

FROM source AS worker-build
RUN --mount=type=cache,target=/root/.cache/go-build CGO_ENABLED=0 GOOS=linux go build -a -o video-upscaler .


# ========================================================= #

FROM scratch AS cli
COPY --link --from=cli-build /workspace/video-upscaler /usr/bin/video-upscaler
ENTRYPOINT [ "/usr/bin/video-upscaler" ]

FROM golang:1.20-alpine AS tini
ARG TINI_VERSION=v0.19.0
ADD https://github.com/krallin/tini/releases/download/${TINI_VERSION}/tini /tini
RUN chmod +x /tini


FROM abihf/video-upscaler:base AS worker
COPY --from=tini /tini /tini
ENTRYPOINT ["/tini", "--"]
COPY script.vpy models /upscale/
COPY --link --from=worker-build /workspace/video-upscaler /usr/bin/video-upscaler
CMD [ "/usr/bin/video-upscaler", "worker" ]

