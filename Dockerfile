ARG BASE_IMAGE=ghcr.io/abihf/video-upscaler:base

# GO builder
FROM golang:1.21rc2-alpine AS source
WORKDIR /workspace


FROM source AS worker-build
RUN --mount=type=cache,target=/root/.cache/go-build \
  --mount=type=cache,target=/go/pkg \
  --mount=type=bind,target=. \
  CGO_ENABLED=0 GOOS=linux GOAMD64=v3 go build -v -o /video-upscaler .

# ========================================================= #

FROM scratch AS cli
COPY --link --from=worker-build /video-upscaler /usr/bin/video-upscaler
ENTRYPOINT [ "/usr/bin/video-upscaler" ]

# ========================================================= #

FROM $BASE_IMAGE AS worker

ARG TINI_VERSION=v0.19.0
ADD --link --chmod=0755 https://github.com/krallin/tini/releases/download/${TINI_VERSION}/tini /tini
ENTRYPOINT ["/tini", "--"]

COPY --link script.py /upscale/
COPY --link --from=worker-build --chmod=0755 /video-upscaler /usr/bin/video-upscaler
CMD [ "/usr/bin/video-upscaler", "worker" ]
