#syntax=docker/dockerfile:1-labs

# base image with prepared apt
# it also re-enable apt cache
FROM ubuntu:24.04 AS base
ARG ENABLE_APT_CACHE=1
RUN --mount=target=/var/cache/apt,type=cache,sharing=locked bash -euxo pipefail <<EOF
	sed -i "s|deb [a-z]*://[^ ]* |deb http://linux.domainesia.com/ubuntu/ubuntu-archive/ |g" /etc/apt/sources.list
	test "${ENABLE_APT_CACHE}" = "1" && rm /etc/apt/apt.conf.d/docker-clean || true
	apt-get update
	apt-get install -y curl
	curl -LO https://developer.download.nvidia.com/compute/cuda/repos/ubuntu2404/x86_64/cuda-keyring_1.1-1_all.deb
	dpkg -i cuda-keyring_1.1-1_all.deb
	rm cuda-keyring_1.1-1_all.deb
	apt-get update
	apt-get upgrade -y
EOF

# ========================================================= #
FROM base AS dev
WORKDIR /workspace
RUN --mount=target=/var/cache/apt,type=cache,sharing=locked \
	apt-get install -y gcc-14 g++-14 build-essential git pkg-config

FROM scratch AS cflags
COPY <<EOF /cflags.sh
	baseflags=(-march=x86-64-v3 -O2 -pipe -fno-plt -fexceptions -ffast-math
			-Wp,-D_FORTIFY_SOURCE=2 -Wformat -Werror=format-security
			-fstack-clash-protection -fcf-protection -flto)
	export CC=gcc-14
	export CXX=g++-14
	export CFLAGS="\${baseflags[@]}"
	export CXXFLAGS="\${CFLAGS} -Wp,-D_GLIBCXX_ASSERTIONS"
	export LDFLAGS="-Wl,-O1,--sort-common,--as-needed,-z,relro,-z,now,-flto,-fuse-ld=gold"
EOF

# ========================================================= #

FROM dev AS vapoursynth
RUN --mount=target=/var/cache/apt,type=cache,sharing=locked \
	apt-get install -y python3-dev autoconf automake libtool libzimg-dev libglib2.0-dev python3-pip

COPY --link --from=cflags /cflags.sh .

ARG VAPOURSYNTH_VER=R65
ADD --link  https://github.com/vapoursynth/vapoursynth/archive/${VAPOURSYNTH_VER}.tar.gz vapoursynth-${VAPOURSYNTH_VER}.tar.gz

RUN bash -euxo pipefail <<EOF
  pip install -U --break-system-packages cython setuptools
	tar xf vapoursynth-${VAPOURSYNTH_VER}.tar.gz
	(
		source cflags.sh
		cd vapoursynth-${VAPOURSYNTH_VER}
		./autogen.sh
		./configure --prefix=/usr
		make -s -j8 
		make DESTDIR=/build/vapoursynth install 
		mv /build/vapoursynth/usr/lib/python3.12/site-packages/vapoursynth.so /build/vapoursynth/usr/lib/python3.12 
	)
	rm -rf vapoursynth-${VAPOURSYNTH_VER}
EOF

# ========================================================= #

FROM dev AS ffmpeg
RUN --mount=target=/var/cache/apt,type=cache,sharing=locked \
	apt-get install -y nasm libtheora-dev libopus-dev libvpx-dev libx264-dev libx265-dev libzimg-dev python3-dev

COPY --link --from=cflags /cflags.sh .

ARG NV_CODEC_VER=13.0.19.0
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

ARG FFMPEG_VER=n7.2-dev
ADD --link https://github.com/FFmpeg/FFmpeg/archive/refs/tags/${FFMPEG_VER}.tar.gz ffmpeg-${FFMPEG_VER}.tar.gz

RUN bash -euxo pipefail <<EOF
	tar xf ffmpeg-${FFMPEG_VER}.tar.gz
	(
		source cflags.sh
		cd FFmpeg-${FFMPEG_VER}
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
RUN --mount=target=/var/cache/apt,type=cache,sharing=locked \
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

ARG LSMASH_VER=62a93fb6a23b013d38f30fcb827c2abda69ee3d1
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

ARG LSMASH_PLUGIN_VER=1194.0.0.0
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

FROM dev AS akarin
RUN --mount=target=/var/cache/apt,type=cache,sharing=locked \
	apt-get install -y meson libzimg-dev llvm-15-dev zlib1g-dev libzstd-dev

COPY --link --from=cflags /cflags.sh .

ARG AKARINVS_VER=8b7ff6dcc85bc9935789c799e63f1388dfbd1bd4
ADD --link https://github.com/AkarinVS/vapoursynth-plugin/archive/${AKARINVS_VER}.tar.gz vapoursynth-plugin-${AKARINVS_VER}.tar.gz

COPY --link --from=vapoursynth /build/vapoursynth/ /
RUN bash -euxo pipefail <<EOF
	tar xf vapoursynth-plugin-${AKARINVS_VER}.tar.gz
	(
		source cflags.sh
		cd vapoursynth-plugin-${AKARINVS_VER}
		meson setup --prefix=/usr build -Dstatic-llvm=true
		meson compile -C build
		meson install --destdir /build/akarin -C build
	)
	rm -rf vapoursynth-plugin-${AKARINVS_VER}
EOF

# ========================================================= #

FROM dev AS vsmlrt
RUN --mount=target=/var/cache/apt,type=cache,sharing=locked \
	apt-get install -y cmake cuda-nvcc-12-8 cuda-cudart-dev-12-8 tensorrt-dev

COPY --link --from=cflags /cflags.sh .

ARG VSMLRT_VER=b3d6425d45f688c12946741cb044f7719f64abd8
ADD --link --keep-git-dir=true https://github.com/AmusementClub/vs-mlrt.git#${VSMLRT_VER} vs-mlrt

COPY --link --from=vapoursynth /build/vapoursynth/ /

RUN bash -euxo pipefail <<EOF
	mkdir vs-mlrt/vstrt/build
	(
		source cflags.sh
		cd vs-mlrt/vstrt/build
		cmake .. \
			-D VAPOURSYNTH_INCLUDE_DIRECTORY=/usr/include/vapoursynth \
			-D CMAKE_CXX_COMPILER=g++-14 \
			-D CMAKE_CXX_FLAGS="$CXXFLAGS" \
			-D TENSORRT_HOME="/opt/TensorRT"
		make
	)
	mkdir -p /build/vsmlrt/usr/lib/vapoursynth/vsmlrt-cuda
	mkdir -p /build/vsmlrt/usr/lib/python3/dist-packages/
	cp vs-mlrt/vstrt/build/libvstrt.so /build/vsmlrt/usr/lib/vapoursynth
	cp vs-mlrt/scripts/vsmlrt.py /build/vsmlrt/usr/lib/python3/dist-packages/
	ln -sf /models /build/vsmlrt/usr/lib/vapoursynth/models
	ln -sf /usr/src/tensorrt/bin/trtexec /build/vsmlrt/usr/lib/vapoursynth/vsmlrt-cuda/
EOF

# ========================================================= #

FROM base AS runtime
RUN --mount=target=/var/cache/apt,type=cache,sharing=locked \
	apt-get install -y --no-install-recommends \
	libpython3.12t64  \
	libtheora0 libzimg2 libopus0 libvpx9 libx264-164 libx265-199 libnvinfer-bin tensorrt-libs cuda-cudart-12-8
COPY --link --from=ffmpeg /build/ffmpeg/ /
COPY --link --from=vapoursynth /build/vapoursynth/ /
COPY --link --from=lsmash /build/lsmash/ /
COPY --link --from=akarin /build/akarin/ /
COPY --link --from=vsmlrt /build/vsmlrt/ /
VOLUME [ "/models" ]
ENV NVIDIA_DRIVER_CAPABILITIES=all
