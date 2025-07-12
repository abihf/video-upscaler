# base image with prepared apt
# it also re-enable apt cache
FROM ubuntu:24.04 AS base
ARG ENABLE_APT_CACHE=1
RUN --mount=target=/var/cache/apt,type=cache,sharing=locked bash -euxo pipefail <<EOF
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
	apt-get install -y build-essential git pkg-config gpg
COPY <<EOF /etc/apt/sources.list.d/llvm.list
deb https://apt.llvm.org/noble/ llvm-toolchain-noble-20 main
EOF
RUN --mount=target=/var/cache/apt,type=cache,sharing=locked \
	curl -fsSL https://apt.llvm.org/llvm-snapshot.gpg.key | apt-key add - && \
	apt-get update && \
	apt-get install -y clang-20 clang++-20 lld-20

FROM scratch AS cflags
COPY <<EOF /cflags.sh
	baseflags=(-march=x86-64-v3 -O2 -pipe -fno-plt -fexceptions -ffast-math
			-Wp,-D_FORTIFY_SOURCE=2 -Wformat -Werror=format-security
			-fstack-clash-protection -fcf-protection -flto)
	export CC=clang-20
	export CXX=clang++-20
	export CFLAGS="\${baseflags[@]}"
	export CXXFLAGS="\${CFLAGS} -Wp,-D_GLIBCXX_ASSERTIONS"
	export LDFLAGS="-Wl,-O1,--sort-common,--as-needed,-z,relro,-z,now,-flto,-fuse-ld=lld-20"
EOF

# ========================================================= #

FROM dev AS vapoursynth
RUN --mount=target=/var/cache/apt,type=cache,sharing=locked \
	apt-get install -y python3-dev autoconf automake libtool libzimg-dev libglib2.0-dev python3-pip

COPY --link --from=cflags /cflags.sh .

ARG VAPOURSYNTH_VER=R72
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
	apt-get install -y nasm libtheora-dev libopus-dev libvpx-dev libzimg-dev python3-dev

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

ARG FFMPEG_VER=n7.1.1
ADD --link https://github.com/FFmpeg/FFmpeg/archive/refs/tags/${FFMPEG_VER}.tar.gz ffmpeg-${FFMPEG_VER}.tar.gz

RUN bash -euxo pipefail <<EOF
	tar xf ffmpeg-${FFMPEG_VER}.tar.gz
	(
		source cflags.sh
		cd FFmpeg-${FFMPEG_VER}
		export LDFLAGS="-Wl,-O1,--sort-common,--as-needed,-z,relro,-z,now"
		./configure \
			--cc=clang-20 \
			--cxx=clang++-20 \
			--extra-ldflags="-fuse-ld=lld-20" \
			--prefix=/usr \
			--disable-debug \
			--disable-static \
			--disable-ffplay \
			--disable-doc \
			--enable-lto \
                        --enable-libzimg \
			--enable-libopus \
			--enable-libtheora \
			--enable-libvpx \
			--enable-nvdec \
			--enable-nvenc \
			--enable-shared
		make -s -j$(nproc)
		make DESTDIR=/build/ffmpeg install
	)
	rm -rf ffmpeg-${FFMPEG_VER}
EOF

# ========================================================= #
FROM dev AS bestsource
RUN --mount=target=/var/cache/apt,type=cache,sharing=locked \
	apt-get install -y meson libzimg-dev libxxhash-dev

COPY --link --from=cflags /cflags.sh .
ARG BESTSOURCE_VER=R12
ADD --link --keep-git-dir=true https://github.com/vapoursynth/bestsource.git#${BESTSOURCE_VER} bestsource

COPY --link --from=vapoursynth /build/vapoursynth/ /
COPY --link --from=ffmpeg /build/ffmpeg/ /

RUN bash -euxo pipefail <<EOF
	(
		source cflags.sh
		cd bestsource
		git submodule update --init
		export AR=llvm-ar-20
		meson setup --prefix=/usr build
		meson compile -C build
		meson install --destdir /build/bestsource -C build
	)
EOF

# ========================================================= #

FROM dev AS akarin
RUN --mount=target=/var/cache/apt,type=cache,sharing=locked \
	apt-get install -y meson libzimg-dev llvm-20-dev zlib1g-dev libzstd-dev

COPY --link --from=cflags /cflags.sh .

ARG AKARINVS_VER=1.0.0
ADD --link https://github.com/Jaded-Encoding-Thaumaturgy/akarin-vapoursynth-plugin/archive/v${AKARINVS_VER}.tar.gz vapoursynth-plugin-v${AKARINVS_VER}.tar.gz

COPY --link --from=vapoursynth /build/vapoursynth/ /
RUN bash -euxo pipefail <<EOF
	tar xf vapoursynth-plugin-v${AKARINVS_VER}.tar.gz
	(
		source cflags.sh
		cd akarin-vapoursynth-plugin-${AKARINVS_VER}
		export LLVM_CONFIG=llvm-config-20
		meson setup --prefix=/usr build -Dstatic-llvm=true
		meson compile -C build
		meson install --destdir /build/akarin -C build
	)
	rm -rf akarin-vapoursynth-plugin-${AKARINVS_VER}
EOF

# ========================================================= #

FROM dev AS vsmlrt
RUN --mount=target=/var/cache/apt,type=cache,sharing=locked \
	apt-get install -y cmake cuda-nvcc-12-9 cuda-cudart-dev-12-9 libnvinfer-dev

COPY --link --from=cflags /cflags.sh .

ARG VSMLRT_VER=467e4fc62f1eddf65ca7a2a3eeca74b3226cd44e
ADD --link --keep-git-dir=true https://github.com/AmusementClub/vs-mlrt.git#${VSMLRT_VER} vs-mlrt

COPY --link --from=vapoursynth /build/vapoursynth/ /

RUN bash -euxo pipefail <<EOF
	mkdir vs-mlrt/vstrt/build
	(
		source cflags.sh
		cd vs-mlrt/vstrt/build
		cmake .. \
			-D VAPOURSYNTH_INCLUDE_DIRECTORY=/usr/include/vapoursynth \
			-D CMAKE_CXX_COMPILER=clang++-20 \
			-D CMAKE_CXX_FLAGS="\$CXXFLAGS" \
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
	libpython3.12t64 libtheora0 libzimg2 libopus0 libvpx9 libxxhash0 \
	libnvinfer-bin libnvinfer10 cuda-cudart-12-9
COPY --link --from=ffmpeg /build/ffmpeg/ /
COPY --link --from=vapoursynth /build/vapoursynth/ /
COPY --link --from=bestsource /build/bestsource/ /
# COPY --link --from=akarin /build/akarin/ /
COPY --link --from=vsmlrt /build/vsmlrt/ /
VOLUME [ "/models" ]
ENV NVIDIA_DRIVER_CAPABILITIES=all
