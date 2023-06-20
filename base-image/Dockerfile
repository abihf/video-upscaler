FROM ghcr.io/stargz-containers/ubuntu:22.04-esgz as base
RUN --mount=target=/var/lib/apt/lists,type=cache,sharing=locked \
    --mount=target=/var/cache/apt,type=cache,sharing=locked \
    apt-get update && apt-get install -y curl

FROM base as tensorrt
RUN --mount=target=/var/lib/apt/lists,type=cache,sharing=locked \
    --mount=target=/var/cache/apt,type=cache,sharing=locked \
    curl -LO https://developer.download.nvidia.com/compute/cuda/repos/ubuntu2204/x86_64/cuda-keyring_1.1-1_all.deb && \
    dpkg -i cuda-keyring_1.1-1_all.deb && \
    rm cuda-keyring_1.1-1_all.deb && \
    apt-get update && \
    apt-get install -y tensorrt-libs

# ========================================================= #
FROM tensorrt AS dev
WORKDIR /workspace
RUN --mount=target=/var/lib/apt/lists,type=cache,sharing=locked \
    --mount=target=/var/cache/apt,type=cache,sharing=locked \
    apt-get install -y build-essential git pkg-config


# ========================================================= #

FROM dev AS ffmpeg
RUN --mount=target=/var/lib/apt/lists,type=cache,sharing=locked \
    --mount=target=/var/cache/apt,type=cache,sharing=locked \
    apt-get install -y nasm libtheora-dev libopus-dev libvpx-dev libx264-dev libx265-dev

ARG NV_CODEC_VER=12.0.16.0
ADD --link https://github.com/FFmpeg/nv-codec-headers/releases/download/n${NV_CODEC_VER}/nv-codec-headers-${NV_CODEC_VER}.tar.gz .

RUN tar xf nv-codec-headers-${NV_CODEC_VER}.tar.gz && \ 
    cd nv-codec-headers-${NV_CODEC_VER} && \
    make PREFIX=/usr install && \
    cd .. && rm -rf nv-codec-headers-${NV_CODEC_VER}

ARG FFMPEG_VER=6.0
ADD --link https://ffmpeg.org/releases/ffmpeg-${FFMPEG_VER}.tar.xz .

RUN tar xf ffmpeg-${FFMPEG_VER}.tar.xz && \
    cd ffmpeg-${FFMPEG_VER} && \
    ./configure \
    --prefix=/usr \
    --disable-debug \
    --disable-static \
    --enable-gpl \
    --enable-lto \
    --enable-libopus \
    --enable-libtheora \
    --enable-libvpx \
    --enable-libx264 \
    --enable-libx265 \
    --enable-nvdec \
    --enable-nvenc \
    --enable-shared \
    --enable-version3 && \
    make -j8 && \
    make DESTDIR=/build/ffmpeg install && \
    cd .. && rm -rf ffmpeg-${FFMPEG_VER}


# ========================================================= #

FROM dev AS vapoursynth
RUN --mount=target=/var/lib/apt/lists,type=cache,sharing=locked \
    --mount=target=/var/cache/apt,type=cache,sharing=locked \
    apt-get install -y python3-dev autoconf automake libtool libzimg-dev libglib2.0-dev cython3

ARG VAPOURSYNTH_VER=R63
ADD --link  https://github.com/vapoursynth/vapoursynth/archive/${VAPOURSYNTH_VER}.tar.gz vapoursynth-${VAPOURSYNTH_VER}.tar.gz

RUN tar xf vapoursynth-${VAPOURSYNTH_VER}.tar.gz && \
    cd vapoursynth-${VAPOURSYNTH_VER} && ./autogen.sh \
    && ./configure --prefix=/usr \
    && make -j8 \
    && make DESTDIR=/build/vapoursynth install \
    && mv /build/vapoursynth/usr/lib/python3.10/site-packages/vapoursynth.so /build/vapoursynth/usr/lib/python3.10 \
    && cd .. && rm -rf vapoursynth-${VAPOURSYNTH_VER}


# ========================================================= #

FROM dev AS lsmash

RUN --mount=target=/var/lib/apt/lists,type=cache,sharing=locked \
    --mount=target=/var/cache/apt,type=cache,sharing=locked \
    apt-get install -y meson libzimg-dev

ARG OBUPARSE_VER=055be27797cc37cf6e0c3c0bafc1aa752608b7fb
ADD --link https://github.com/dwbuiten/obuparse/archive/${OBUPARSE_VER}.tar.gz obuparse-${OBUPARSE_VER}.tar.gz

RUN tar xf obuparse-${OBUPARSE_VER}.tar.gz && \
    cd obuparse-${OBUPARSE_VER} && \
    make && \
    make install && \
    mkdir -p /build/lsmash/usr/local/lib && \
    mkdir -p /build/lsmash/usr/local/include && \
    make install PREFIX=/build/lsmash/usr/local

ARG LSMASH_VER=2c0696caacb91af2737b94b51738d5dfcea37dfe
ADD --link https://github.com/vimeo/l-smash/archive/${LSMASH_VER}.tar.gz l-smash-${LSMASH_VER}.tar.gz

COPY --link --from=ffmpeg /build/ffmpeg/ /
RUN tar xf l-smash-${LSMASH_VER}.tar.gz \
    && cd l-smash-${LSMASH_VER} \
    && ./configure --prefix=/usr --enable-shared --disable-static \
    && make \
    && make install \
    && make DESTDIR=/build/lsmash install-lib

ARG LSMASH_PLUGIN_VER=20230611
ADD --link https://github.com/HomeOfAviSynthPlusEvolution/L-SMASH-Works/archive/${LSMASH_PLUGIN_VER}.tar.gz L-SMASH-Works-${LSMASH_PLUGIN_VER}.tar.gz

COPY --link --from=vapoursynth /build/vapoursynth/ /
RUN tar xf L-SMASH-Works-${LSMASH_PLUGIN_VER}.tar.gz && \
    cd L-SMASH-Works-${LSMASH_PLUGIN_VER}/VapourSynth && \
    meson setup --prefix=/usr build && \
    meson compile -C build && \
    meson install --destdir /build/lsmash -C build

# ========================================================= #

FROM dev AS vsmlrt
RUN --mount=target=/var/lib/apt/lists,type=cache,sharing=locked \
    --mount=target=/var/cache/apt,type=cache,sharing=locked \
    apt-get install -y cmake tensorrt-dev cuda-nvcc-12-1 libnvinfer-samples cuda-profiler-api-12-1

# rebuild trtexec
# somehow provided package doesn't work
RUN cd /usr/src/tensorrt/samples/trtexec && \
    make && \
    mkdir -p /build/vsmlrt/usr/lib/vapoursynth/vsmlrt-cuda && \
    cp /usr/src/tensorrt/bin/trtexec /build/vsmlrt/usr/lib/vapoursynth/vsmlrt-cuda/trtexec && \
    cp /usr/local/cuda/targets/x86_64-linux/lib/libcudart.so* /build/vsmlrt/usr/lib

ARG VSMLRT_VER=cf2bfbffe10361990e2cdcaada78c6a34046c8e6
RUN git clone https://github.com/AmusementClub/vs-mlrt \
    && cd vs-mlrt \
    && git checkout ${VSMLRT_VER}

COPY --link --from=vapoursynth /build/vapoursynth/ /

RUN mkdir vs-mlrt/vstrt/build \
    && cd vs-mlrt/vstrt/build && cmake .. -D USE_NVINFER_PLUGIN=ON -D VAPOURSYNTH_INCLUDE_DIRECTORY=/usr/include/vapoursynth \
    && make
RUN mkdir -p /build/vsmlrt/usr/lib/vapoursynth && \
    cp vs-mlrt/vstrt/build/libvstrt.so /build/vsmlrt/usr/lib/vapoursynth && \
    ln -sf /models /build/vsmlrt/usr/lib/vapoursynth/models && \
    mkdir -p /build/vsmlrt/usr/lib/python3/dist-packages/ && \
    cp vs-mlrt/scripts/vsmlrt.py /build/vsmlrt/usr/lib/python3/dist-packages/


# ========================================================= #

FROM tensorrt AS runtime
RUN --mount=target=/var/lib/apt/lists,type=cache,sharing=locked \
    --mount=target=/var/cache/apt,type=cache,sharing=locked \
    apt-get install -y libtheora0 libzimg2 libopus0 libvpx7 libx264-163 libx265-199 libnvinfer-bin libpython3.10 rsync
COPY --link --from=ffmpeg /build/ffmpeg/ /
COPY --link --from=vapoursynth /build/vapoursynth/ /
COPY --link --from=lsmash /build/lsmash/ /
COPY --link --from=vsmlrt /build/vsmlrt/ /
VOLUME [ "/models" ]
