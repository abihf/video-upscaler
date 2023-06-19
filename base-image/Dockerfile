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
ARG FFMPEG_VER=6.0
RUN curl -LO https://ffmpeg.org/releases/ffmpeg-${FFMPEG_VER}.tar.xz
RUN tar xvf ffmpeg-${FFMPEG_VER}.tar.xz

RUN curl -LO https://github.com/FFmpeg/nv-codec-headers/releases/download/n12.0.16.0/nv-codec-headers-12.0.16.0.tar.gz && \
    tar xvf nv-codec-headers-12.0.16.0.tar.gz
RUN cd nv-codec-headers-12.0.16.0 && \
    make PREFIX=/usr install

RUN --mount=target=/var/lib/apt/lists,type=cache,sharing=locked \
    --mount=target=/var/cache/apt,type=cache,sharing=locked \
    apt-get install -y nasm libtheora-dev libopus-dev libvpx-dev libx264-dev libx265-dev
RUN cd ffmpeg-${FFMPEG_VER} && \
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
    --enable-version3
RUN cd ffmpeg-${FFMPEG_VER} && \
    make -j8 && \
    make DESTDIR=/build/ffmpeg install


# ========================================================= #

FROM dev AS vapoursynth
ARG VAPOURSYNTH_VER=R62
RUN --mount=target=/var/lib/apt/lists,type=cache,sharing=locked \
    --mount=target=/var/cache/apt,type=cache,sharing=locked \
    apt-get install -y python3-dev autoconf automake libtool libzimg-dev libglib2.0-dev cython3

RUN curl -L https://github.com/vapoursynth/vapoursynth/archive/${VAPOURSYNTH_VER}.tar.gz | tar xvz

RUN cd vapoursynth-${VAPOURSYNTH_VER} && ./autogen.sh \
    && ./configure --prefix=/usr \
    && make -j8 \
    && make DESTDIR=/build/vapoursynth install \
    && mv /build/vapoursynth/usr/lib/python3.10/site-packages/vapoursynth.so /build/vapoursynth/usr/lib/python3.10


# ========================================================= #

FROM dev AS lsmash

RUN --mount=target=/var/lib/apt/lists,type=cache,sharing=locked \
    --mount=target=/var/cache/apt,type=cache,sharing=locked \
    apt-get install -y meson libzimg-dev

COPY --from=ffmpeg /build/ffmpeg/ /

ARG LSMASH_VER=2.14.5
RUN curl -L https://github.com/l-smash/l-smash/archive/v${LSMASH_VER}.tar.gz | tar xvz \
    && cd l-smash-${LSMASH_VER} \
    && ./configure --prefix=/usr --enable-shared --disable-static \
    && make \
    && make install \
    && make DESTDIR=/build/lsmash install-lib

ARG LSMASH_PLUGIN_VER=fb891d030c26af63abf19ab098638a40e8a1d27b
RUN git clone https://github.com/AkarinVS/L-SMASH-Works \
    && cd L-SMASH-Works/VapourSynth \
    && git checkout ${LSMASH_PLUGIN_VER}

COPY --from=vapoursynth /build/vapoursynth/ /

RUN cd L-SMASH-Works/VapourSynth && \
    meson setup --prefix=/usr build && \
    meson compile -C build && \
    meson install --destdir /build/lsmash -C build


# ========================================================= #

FROM dev AS vsmlrt
RUN --mount=target=/var/lib/apt/lists,type=cache,sharing=locked \
    --mount=target=/var/cache/apt,type=cache,sharing=locked \
    apt-get install -y cmake tensorrt-dev cuda-nvcc-12-1

COPY --from=vapoursynth /build/vapoursynth/ /

ARG VSMLRT_VER=cf2bfbffe10361990e2cdcaada78c6a34046c8e6
RUN git clone https://github.com/AmusementClub/vs-mlrt \
    && mkdir vs-mlrt/vstrt/build \
    && cd vs-mlrt/vstrt/build \
    && git checkout ${VSMLRT_VER}

RUN cd vs-mlrt/vstrt/build && cmake .. -D USE_NVINFER_PLUGIN=ON -D VAPOURSYNTH_INCLUDE_DIRECTORY=/usr/include/vapoursynth \
    && make
RUN mkdir -p /build/vsmlrt/usr/lib/vapoursynth && \
    cp vs-mlrt/vstrt/build/libvstrt.so /build/vsmlrt/usr/lib/vapoursynth && \
    ln -sf /models /build/vsmlrt/usr/lib/vapoursynth/models && \
    mkdir -p /build/vsmlrt/usr/lib/python3/dist-packages/ && \
    cp vs-mlrt/scripts/vsmlrt.py /build/vsmlrt/usr/lib/python3/dist-packages/

RUN --mount=target=/var/lib/apt/lists,type=cache,sharing=locked \
    --mount=target=/var/cache/apt,type=cache,sharing=locked \
    apt-get install -y libnvinfer-samples cuda-profiler-api-12-1

RUN cd /usr/src/tensorrt/samples/trtexec && \
    make && \
    mkdir -p /build/vsmlrt/usr/lib/vapoursynth/vsmlrt-cuda && \
    cp /usr/src/tensorrt/bin/trtexec /build/vsmlrt/usr/lib/vapoursynth/vsmlrt-cuda/trtexec && \
    cp /usr/local/cuda/targets/x86_64-linux/lib/libcudart.so* /build/vsmlrt/usr/lib


# ========================================================= #

FROM tensorrt AS runtime
RUN --mount=target=/var/lib/apt/lists,type=cache,sharing=locked \
    --mount=target=/var/cache/apt,type=cache,sharing=locked \
    apt-get install -y libtheora0 libzimg2 libopus0 libvpx7 libx264-163 libx265-199 libnvinfer-bin libpython3.10 rsync
COPY --from=ffmpeg /build/ffmpeg/ /
COPY --from=vapoursynth /build/vapoursynth/ /
COPY --from=lsmash /build/lsmash/ /
COPY --from=vsmlrt /build/vsmlrt/ /
VOLUME [ "/models" ]
