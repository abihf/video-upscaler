#syntax=docker/dockerfile:1-labs

# base image with prepared apt
# it also re-enable apt cache
FROM ubuntu:22.04 as base
ARG ENABLE_APT_CACHE=1
RUN test "${ENABLE_APT_CACHE}" = "1" && rm /etc/apt/apt.conf.d/docker-clean || true
RUN --mount=target=/var/lib/apt/lists,type=cache,sharing=locked \
	--mount=target=/var/cache/apt,type=cache,sharing=locked \
	apt-get update && apt-get install -y curl && \
	curl -LO https://developer.download.nvidia.com/compute/cuda/repos/ubuntu2204/x86_64/cuda-keyring_1.1-1_all.deb && \
	dpkg -i cuda-keyring_1.1-1_all.deb && \
	rm cuda-keyring_1.1-1_all.deb && \
	apt-get update

# ========================================================= #
FROM base AS dev
WORKDIR /workspace
RUN --mount=target=/var/lib/apt/lists,type=cache,sharing=locked \
	--mount=target=/var/cache/apt,type=cache,sharing=locked \
	apt-get install -y gcc-12 g++-12 build-essential git pkg-config

FROM scratch as cflags
COPY <<EOF /cflags.sh
	baseflags=(-march=x86-64-v3 -O2 -pipe -fno-plt -fexceptions
			-Wp,-D_FORTIFY_SOURCE=2 -Wformat -Werror=format-security
			-fstack-clash-protection -fcf-protection -flto)
	export CC=gcc-12
	export CXX=g++-12
	export CFLAGS="\${baseflags[@]}"
	export CXXFLAGS="\${CFLAGS} -Wp,-D_GLIBCXX_ASSERTIONS"
	export LDFLAGS="-Wl,-O1,--sort-common,--as-needed,-z,relro,-z,now,-flto,-fuse-ld=gold"
EOF

# ========================================================= #

FROM dev AS vapoursynth
RUN --mount=target=/var/lib/apt/lists,type=cache,sharing=locked \
	--mount=target=/var/cache/apt,type=cache,sharing=locked \
	apt-get install -y python3-dev autoconf automake libtool libzimg-dev libglib2.0-dev python3-pip

COPY --link --from=cflags /cflags.sh .

ARG VAPOURSYNTH_VER=R65
ADD --link  https://github.com/vapoursynth/vapoursynth/archive/${VAPOURSYNTH_VER}.tar.gz vapoursynth-${VAPOURSYNTH_VER}.tar.gz

RUN bash -euxo pipefail <<EOF
  pip install -U cython setuptools wheel
	tar xf vapoursynth-${VAPOURSYNTH_VER}.tar.gz
	(
		source cflags.sh
		cd vapoursynth-${VAPOURSYNTH_VER}
		./autogen.sh
		./configure --prefix=/usr
		make -s -j8 
		make DESTDIR=/build/vapoursynth install 
		mv /build/vapoursynth/usr/lib/python3.10/site-packages/vapoursynth.so /build/vapoursynth/usr/lib/python3.10 
	)
	rm -rf vapoursynth-${VAPOURSYNTH_VER}
EOF

# ========================================================= #

FROM dev AS ffmpeg
RUN --mount=target=/var/lib/apt/lists,type=cache,sharing=locked \
	--mount=target=/var/cache/apt,type=cache,sharing=locked \
	apt-get install -y nasm libtheora-dev libopus-dev libvpx-dev libx264-dev libx265-dev libzimg-dev python3-dev

COPY --link --from=cflags /cflags.sh .

ARG NV_CODEC_VER=12.1.14.0
ADD --link https://github.com/FFmpeg/nv-codec-headers/releases/download/n${NV_CODEC_VER}/nv-codec-headers-${NV_CODEC_VER}.tar.gz .

RUN bash <<EOF
	set -euxo pipefail
	tar xf nv-codec-headers-${NV_CODEC_VER}.tar.gz
	(
		source cflags.sh
		cd nv-codec-headers-${NV_CODEC_VER}
		make PREFIX=/usr install
	)
	rm -rf nv-codec-headers-${NV_CODEC_VER}
EOF

ARG FFMPEG_VER=6.1.1
ADD --link https://ffmpeg.org/releases/ffmpeg-${FFMPEG_VER}.tar.xz .

RUN bash <<EOF
	set -euxo pipefail
	tar xf ffmpeg-${FFMPEG_VER}.tar.xz
	(
		source cflags.sh
		cd ffmpeg-${FFMPEG_VER}
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
		make -s -j8
		make DESTDIR=/build/ffmpeg install
	)
	rm -rf ffmpeg-${FFMPEG_VER}
EOF

# ========================================================= #

FROM dev AS lsmash
RUN --mount=target=/var/lib/apt/lists,type=cache,sharing=locked \
	--mount=target=/var/cache/apt,type=cache,sharing=locked \
	apt-get install -y meson libzimg-dev

COPY --link --from=cflags /cflags.sh .

ARG OBUPARSE_VER=478adcc872d5a8a19443e87910508e261a0a60ba
ADD --link https://github.com/dwbuiten/obuparse/archive/${OBUPARSE_VER}.tar.gz obuparse-${OBUPARSE_VER}.tar.gz

RUN bash <<EOF
	set -euxo pipefail
	tar xf obuparse-${OBUPARSE_VER}.tar.gz
	(
		source cflags.sh
		cd obuparse-${OBUPARSE_VER}
		make -s
		make install
		mkdir -p /build/lsmash/usr/lib
		mkdir -p /build/lsmash/usr/include
		make install PREFIX=/build/lsmash/usr
	)
	rm -rf obuparse-${OBUPARSE_VER}
EOF

ARG LSMASH_VER=2c0696caacb91af2737b94b51738d5dfcea37dfe
ADD --link https://github.com/vimeo/l-smash/archive/${LSMASH_VER}.tar.gz l-smash-${LSMASH_VER}.tar.gz

COPY --link --from=ffmpeg /build/ffmpeg/ /
RUN bash <<EOF
	set -euxo pipefail
	tar xf l-smash-${LSMASH_VER}.tar.gz
	(
		source cflags.sh
		cd l-smash-${LSMASH_VER} 
		./configure --prefix=/usr --enable-shared --disable-static 
		make -s
		make install 
		make DESTDIR=/build/lsmash install-lib
	)
	rm -rf l-smash-${LSMASH_VER}
EOF

ARG LSMASH_PLUGIN_VER=1167.0.0.0
ADD --link https://github.com/HomeOfAviSynthPlusEvolution/L-SMASH-Works/archive/${LSMASH_PLUGIN_VER}.tar.gz L-SMASH-Works-${LSMASH_PLUGIN_VER}.tar.gz

COPY --link --from=vapoursynth /build/vapoursynth/ /
RUN bash <<EOF
	set -euxo pipefail
	tar xf L-SMASH-Works-${LSMASH_PLUGIN_VER}.tar.gz
	(
		source cflags.sh
		cd L-SMASH-Works-${LSMASH_PLUGIN_VER}/VapourSynth
		meson setup --prefix=/usr build
		meson compile -C build
		meson install --destdir /build/lsmash -C build
	)
	rm -rf L-SMASH-Works-${LSMASH_PLUGIN_VER}
EOF

# ========================================================= #

FROM base AS tensorrt
ARG TENSORRT_BASE_VER=9.3.0
ARG TENSORRT_VER=${TENSORRT_BASE_VER}.1
RUN curl -L https://developer.nvidia.com/downloads/compute/machine-learning/tensorrt/${TENSORRT_BASE_VER}/tensorrt-${TENSORRT_VER}.linux.x86_64-gnu.cuda-12.2.tar.gz | tar -x -z -C /opt/ \
	&& mv /opt/TensorRT-${TENSORRT_VER} /opt/TensorRT && \
	rm /opt/TensorRT/lib/*.a /opt/TensorRT/lib/*/*.a

FROM dev AS vsmlrt
RUN sleep 2
RUN --mount=target=/var/lib/apt/lists,type=cache,sharing=locked \
	--mount=target=/var/cache/apt,type=cache,sharing=locked \
	apt-get install -y cmake cuda-nvcc-12-3 cuda-cudart-dev-12-3 

COPY --link --from=cflags /cflags.sh .

ARG VSMLRT_VER=48f57dac7ccec47ed3b52f5eeb740d8121083fe9
ADD --link --keep-git-dir=true https://github.com/AmusementClub/vs-mlrt.git#${VSMLRT_VER} vs-mlrt

COPY --link --from=vapoursynth /build/vapoursynth/ /

RUN --mount=type=bind,from=tensorrt,source=/opt/TensorRT,target=/opt/TensorRT \
	bash -euxo pipefail <<EOF
	mkdir vs-mlrt/vstrt/build
	(
		source cflags.sh
		cd vs-mlrt/vstrt/build
		cmake .. \
			-D VAPOURSYNTH_INCLUDE_DIRECTORY=/usr/include/vapoursynth \
			-D CMAKE_CXX_COMPILER=g++-12 \
			-D CMAKE_CXX_FLAGS="$CXXFLAGS" \
			-D TENSORRT_HOME="/opt/TensorRT"
		make
	)
	mkdir -p /build/vsmlrt/usr/lib/vapoursynth
	mkdir -p /build/vsmlrt/usr/lib/python3/dist-packages/
	cp vs-mlrt/vstrt/build/libvstrt.so /build/vsmlrt/usr/lib/vapoursynth
	cp vs-mlrt/scripts/vsmlrt.py /build/vsmlrt/usr/lib/python3/dist-packages/
	ln -sf /models /build/vsmlrt/usr/lib/vapoursynth/models
EOF

# ========================================================= #

FROM base AS runtime
RUN sleep 5
RUN --mount=target=/var/lib/apt/lists,type=cache,sharing=locked \
	--mount=target=/var/cache/apt,type=cache,sharing=locked \
	apt-get install -y --no-install-recommends \
	libpython3.10  \
	libtheora0 libzimg2 libopus0 libvpx7 libx264-163 libx265-199
COPY --link --from=ffmpeg /build/ffmpeg/ /
COPY --link --from=vapoursynth /build/vapoursynth/ /
COPY --link --from=lsmash /build/lsmash/ /
COPY --link --from=vsmlrt /build/vsmlrt/ /
COPY --link --from=tensorrt /opt/TensorRT/lib/ /usr/lib/
COPY --link --from=tensorrt /opt/TensorRT/bin/trtexec /usr/lib/vapoursynth/vsmlrt-cuda/trtexec
VOLUME [ "/models" ]
ENV NVIDIA_DRIVER_CAPABILITIES=all
