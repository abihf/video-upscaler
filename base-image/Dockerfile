#syntax=docker/dockerfile:1-labs

# base image with prepared apt
# it also re-enable apt cache
FROM ubuntu:20.04 as base
ARG ENABLE_APT_CACHE=1
RUN test "${ENABLE_APT_CACHE}" = "1" && rm /etc/apt/apt.conf.d/docker-clean
RUN --mount=target=/var/lib/apt/lists,type=cache,sharing=locked \
    --mount=target=/var/cache/apt,type=cache,sharing=locked \
    apt-get update && apt-get install -y curl gnupg2 software-properties-common && \
    mkdir -p /etc/apt/keyrings/ && \
    curl -L https://repo.radeon.com/rocm/rocm.gpg.key | gpg --dearmor > /etc/apt/keyrings/rocm.gpg && \
    echo "deb [arch=amd64 signed-by=/etc/apt/keyrings/rocm.gpg] https://repo.radeon.com/rocm/apt/5.4 ubuntu main" >> /etc/apt/sources.list.d/rocm.list && \
    echo -e 'Package: *\nPin: release o=repo.radeon.com\nPin-Priority: 600' > tee /etc/apt/preferences.d/rocm-pin-600 && \
    add-apt-repository ppa:ubuntu-toolchain-r/test
ENV DEBIAN_FRONTEND noninteractive

# ========================================================= #
FROM base AS dev
WORKDIR /workspace
RUN --mount=target=/var/lib/apt/lists,type=cache,sharing=locked \
    --mount=target=/var/cache/apt,type=cache,sharing=locked \
    apt-get install -y build-essential gcc-13 g++-13 libstdc++-13-dev git pkg-config

FROM scratch as cflags
COPY <<EOF /cflags.sh
	export CC=/usr/bin/gcc-13
	export CXX=/usr/bin/g++-13
	baseflags=(-march=x86-64-v3 -O2 -pipe -fno-plt -fexceptions
			-Wp,-D_FORTIFY_SOURCE=2 -Wformat -Werror=format-security
			-fstack-clash-protection -fcf-protection)
	export CFLAGS="\${baseflags[@]}"
	export CXXFLAGS="\${CFLAGS} -Wp,-D_GLIBCXX_ASSERTIONS"
	export LDFLAGS="-Wl,-O1,--sort-common,--as-needed,-z,relro,-z,now"
EOF

# ========================================================= #

FROM dev AS vapoursynth
RUN --mount=target=/var/lib/apt/lists,type=cache,sharing=locked \
    --mount=target=/var/cache/apt,type=cache,sharing=locked \
    apt-get install -y python3-dev autoconf automake libtool libzimg-dev libglib2.0-dev cython3

COPY --link --from=cflags /cflags.sh .

ARG VAPOURSYNTH_VER=R63
ADD --link  https://github.com/vapoursynth/vapoursynth/archive/${VAPOURSYNTH_VER}.tar.gz vapoursynth-${VAPOURSYNTH_VER}.tar.gz

RUN bash -euxo pipefail <<EOF
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

ARG NV_CODEC_VER=12.0.16.0
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

ARG FFMPEG_VER=6.0
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

ARG OBUPARSE_VER=f40598f69bb8af14568907dac35661a89fa2342b
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

ARG LSMASH_PLUGIN_VER=1129.0.1.0
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

FROM dev AS vsort
RUN --mount=target=/var/lib/apt/lists,type=cache,sharing=locked \
    --mount=target=/var/cache/apt,type=cache,sharing=locked \
    apt-get install -y  --no-install-recommends \
    rocm-dev migraphx-dev rocblas-dev miopen-hip-dev \
    rocrand-dev rccl-dev hipsparse hipfft hipcub hipblas rocthrust rocm-device-libs 

COPY --link --from=cflags /cflags.sh .

ARG CMAKE_VER=3.27.7
ADD --link https://github.com/Kitware/CMake/releases/download/v${CMAKE_VER}/cmake-${CMAKE_VER}-linux-x86_64.tar.gz cmake.tar.gz
RUN tar -zxf cmake.tar.gz --strip=1 -C /usr

ARG ONNX_VER=v1.16.1
ADD --link --keep-git-dir=true https://github.com/Microsoft/onnxruntime.git#${ONNX_VER} onnxruntime

# RUN bash <<EOF
# 	set -euxo pipefail
# 	(
# 		cd onnxruntime
#     ./build.sh --allow_running_as_root --config Release --parallel --skip_tests --use_migraphx --use_rocm --rocm_home /opt/rocm --rocm_version=5.4
#  	)
# EOF

FROM dev AS vsmlrt
RUN --mount=target=/var/lib/apt/lists,type=cache,sharing=locked \
    --mount=target=/var/cache/apt,type=cache,sharing=locked \
    apt-get install -y cmake tensorrt-dev cuda-nvcc-12-2 libnvinfer-samples cuda-profiler-api-12-2

COPY --link --from=cflags /cflags.sh .

# rebuild trtexec
# somehow provided package doesn't work
RUN bash <<EOF
	set -euxo pipefail
	(
		cd /usr/src/tensorrt/samples/trtexec
		make -s
		mkdir -p /build/vsmlrt/usr/lib/vapoursynth/vsmlrt-cuda
		cp /usr/src/tensorrt/bin/trtexec /build/vsmlrt/usr/lib/vapoursynth/vsmlrt-cuda/trtexec
		cp /usr/local/cuda/targets/x86_64-linux/lib/libcudart.so* /build/vsmlrt/usr/lib
	)
EOF

ARG VSMLRT_VER=54f5d90749fd822e6ac8ab265161d051fd72f3a2
ADD --link --keep-git-dir=true https://github.com/AmusementClub/vs-mlrt.git#${VSMLRT_VER} vs-mlrt

COPY --link --from=vapoursynth /build/vapoursynth/ /

RUN bash <<EOF
	set -euxo pipefail
	mkdir vs-mlrt/vstrt/build
	(
		source cflags.sh
		cd vs-mlrt/vstrt/build
		cmake .. -D USE_NVINFER_PLUGIN=ON -D VAPOURSYNTH_INCLUDE_DIRECTORY=/usr/include/vapoursynth
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
RUN --mount=target=/var/lib/apt/lists,type=cache,sharing=locked \
    --mount=target=/var/cache/apt,type=cache,sharing=locked \
    apt-get install -y --no-install-recommends \
    tensorrt-libs libpython3.10 \
    libtheora0 libzimg2 libopus0 libvpx7 libx264-163 libx265-199
COPY --link --from=ffmpeg /build/ffmpeg/ /
COPY --link --from=vapoursynth /build/vapoursynth/ /
COPY --link --from=lsmash /build/lsmash/ /
COPY --link --from=vsmlrt /build/vsmlrt/ /
VOLUME [ "/models" ]
ENV NVIDIA_DRIVER_CAPABILITIES=all
