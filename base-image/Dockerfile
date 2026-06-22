# base image with prepared apt
# it also re-enable apt cache
FROM ubuntu:26.04 AS base
ARG ENABLE_APT_CACHE=1
RUN --mount=target=/var/cache/apt,type=cache,sharing=locked bash -euxo pipefail <<EOF
	test "${ENABLE_APT_CACHE}" = "1" && rm /etc/apt/apt.conf.d/docker-clean || true
	apt-get update
	apt-get install -y curl gpg
	curl -fsSL https://mkvtoolnix.download/gpg-pub-moritzbunkus.gpg | gpg --dearmor -o /usr/share/keyrings/mkvtoolnix.gpg
	echo 'deb [signed-by=/usr/share/keyrings/mkvtoolnix.gpg] https://mkvtoolnix.download/ubuntu/ resolute main' > /etc/apt/sources.list.d/mkvtoolnix.list
	apt-get update
	apt-get upgrade -y
EOF

# ========================================================= #
FROM base AS dev
WORKDIR /workspace
RUN --mount=target=/var/cache/apt,type=cache,sharing=locked bash -euxo pipefail <<EOF
	curl -fsSL https://apt.llvm.org/llvm-snapshot.gpg.key | gpg --dearmor -o /usr/share/keyrings/llvm-archive-keyring.gpg
	echo 'deb [signed-by=/usr/share/keyrings/llvm-archive-keyring.gpg] https://apt.llvm.org/resolute/ llvm-toolchain-resolute-22 main' > /etc/apt/sources.list.d/llvm.list
	apt-get update
	apt-get install -y build-essential git pkg-config clang-22 clang++-22 lld-22
EOF

# ========================================================= #

FROM dev AS vapoursynth
RUN --mount=target=/var/cache/apt,type=cache,sharing=locked \
	apt-get install -y python3-dev meson libzimg-dev libglib2.0-dev python3-pip

COPY --link cflags.sh .
ARG VAPOURSYNTH_VER=R77
ADD --link https://github.com/vapoursynth/vapoursynth/archive/${VAPOURSYNTH_VER}.tar.gz vapoursynth-${VAPOURSYNTH_VER}.tar.gz

RUN bash -euxo pipefail <<EOF
  pip install -U --break-system-packages cython setuptools
	tar xf vapoursynth-${VAPOURSYNTH_VER}.tar.gz
	(
		source cflags.sh
		cd vapoursynth-${VAPOURSYNTH_VER}
		meson setup --prefix=/usr build
		meson compile -C build
		meson install --destdir /build/vapoursynth -C build
		mkdir -p /build/vapoursynth/usr/include/
		cp include/*.h /build/vapoursynth/usr/include/
		mkdir -p /build/vapoursynth/usr/lib/pkgconfig/
 		mv /build/vapoursynth/usr/lib/python3/dist-packages/vapoursynth/pkgconfig/* /build/vapoursynth/usr/lib/pkgconfig/
		mkdir -p /build/vapoursynth/usr/bin
		ln -sf /usr/lib/python3/dist-packages/vapoursynth/vspipe /build/vapoursynth/usr/bin/vspipe
	)
	rm -rf vapoursynth-${VAPOURSYNTH_VER}
EOF

# ========================================================= #

FROM dev AS ffmpeg
RUN --mount=target=/var/cache/apt,type=cache,sharing=locked \
	apt-get install -y nasm libtheora-dev libopus-dev libvpx-dev libzimg-dev python3-dev

COPY --link cflags.sh .

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

ARG FFMPEG_VER=n8.1.2
ADD --link https://github.com/FFmpeg/FFmpeg/archive/refs/tags/${FFMPEG_VER}.tar.gz ffmpeg-${FFMPEG_VER}.tar.gz

RUN bash -euxo pipefail <<EOF
	tar xf ffmpeg-${FFMPEG_VER}.tar.gz
	(
		source cflags.sh
		cd FFmpeg-${FFMPEG_VER}
		export LDFLAGS="-Wl,-O1,--sort-common,--as-needed,-z,relro,-z,now"
		./configure \
			--cc="\$CC" \
			--cxx="\$CXX" \
			--extra-ldflags="-fuse-ld=lld-22" \
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

COPY --link cflags.sh .
ARG BESTSOURCE_VER=R18
ADD --link --keep-git-dir=true https://github.com/vapoursynth/bestsource.git#${BESTSOURCE_VER} bestsource

COPY --link --from=vapoursynth /build/vapoursynth/ /
COPY --link --from=ffmpeg /build/ffmpeg/ /

RUN bash -euxo pipefail <<EOF
	(
		source cflags.sh
		cd bestsource
		git submodule update --init
		meson setup --prefix=/usr build
		meson compile -C build
		meson install --destdir /build/bestsource -C build
	)
EOF

# ========================================================= #

FROM dev AS akarin
RUN --mount=target=/var/cache/apt,type=cache,sharing=locked \
	apt-get install -y meson libzimg-dev llvm-22-dev zlib1g-dev libzstd-dev

COPY --link cflags.sh .

ARG AKARINVS_VER=1.4.1
ADD --link https://github.com/Jaded-Encoding-Thaumaturgy/akarin-vapoursynth-plugin/archive/v${AKARINVS_VER}.tar.gz vapoursynth-plugin-v${AKARINVS_VER}.tar.gz

COPY --link --from=vapoursynth /build/vapoursynth/ /
RUN bash -euxo pipefail <<EOF
	tar xf vapoursynth-plugin-v${AKARINVS_VER}.tar.gz
	(
		source cflags.sh
		cd akarin-vapoursynth-plugin-${AKARINVS_VER}
		export LLVM_CONFIG=llvm-config-22
		meson setup --prefix=/usr build -Dstatic-llvm=true
		meson compile -C build
		meson install --destdir /build/akarin -C build
	)
	rm -rf akarin-vapoursynth-plugin-${AKARINVS_VER}
EOF

# ========================================================= #

FROM dev AS vsmlrt
ARG CUDA_VER=13-3
RUN --mount=target=/var/cache/apt,type=cache,sharing=locked bash -euxo pipefail <<EOF
	curl -LO https://developer.download.nvidia.com/compute/cuda/repos/ubuntu2604/x86_64/cuda-keyring_1.1-1_all.deb
	dpkg -i cuda-keyring_1.1-1_all.deb
	rm cuda-keyring_1.1-1_all.deb
	apt-get update
	apt-get install --no-install-recommends -y cmake cuda-nvcc-${CUDA_VER}
EOF

COPY --link cflags.sh .

ADD --link https://developer.nvidia.com/downloads/trt/rtx_sdk/secure/1.5/TensorRT-RTX-1.5.0.114-Linux-x86_64-cuda-13.2-Release-external.tar.zst tensorrt.tar.zst

ARG VSMLRT_VER=1a14847b0652271d266efbfb691de15fb04bf988
ADD --link --keep-git-dir=true https://github.com/AmusementClub/vs-mlrt.git#${VSMLRT_VER} vs-mlrt

COPY --link --from=vapoursynth /build/vapoursynth/ /

RUN --mount=target=/var/cache/apt,type=cache,sharing=locked \
	apt-get install -y zstd

RUN bash -euxo pipefail <<EOF
	mkdir -p TensorRT
	tar xf tensorrt.tar.zst --strip-components=1 -C TensorRT
	cp -r TensorRT/lib/* /usr/lib/
	cp -r TensorRT/include/* /usr/include/

	mkdir vs-mlrt/vstrt/build
	(
		source cflags.sh
		cd vs-mlrt/vstrt/build
		cmake .. \
			-D VAPOURSYNTH_INCLUDE_DIRECTORY=/usr/include/vapoursynth \
			-D CMAKE_CXX_COMPILER="\$CXX" \
			-D CMAKE_CXX_FLAGS="\$CXXFLAGS" \
			-D TENSORRT_HOME="/usr"
		make
	)
	mkdir -p /build/vsmlrt/usr/lib/python3/dist-packages/vapoursynth/plugins/vsmlrt-cuda
	cp vs-mlrt/vstrt/build/libvstrt_rtx.so /build/vsmlrt/usr/lib/python3/dist-packages/vapoursynth/plugins
	mv TensorRT/lib/* /build/vsmlrt/usr/lib/
	mv TensorRT/bin/tensorrt_rtx /build/vsmlrt/usr/lib/python3/dist-packages/vapoursynth/plugins/vsmlrt-cuda/
	rm -rf TensorRT
	cp vs-mlrt/scripts/vsmlrt.py /build/vsmlrt/usr/lib/python3/dist-packages/
	ln -sf /models /build/vsmlrt/usr/lib/python3/dist-packages/vapoursynth/plugins/models
EOF


# ========================================================= #

FROM base AS runtime
RUN --mount=target=/var/cache/apt,type=cache,sharing=locked \
	apt-get install -y --no-install-recommends \
	python3.14 libpython3.14 libtheoradec2 libtheoraenc2 libzimg2 libopus0 libvpx12 libxxhash0 \
	mkvtoolnix
COPY --link --from=ffmpeg /build/ffmpeg/ /
COPY --link --from=vapoursynth /build/vapoursynth/ /
COPY --link --from=bestsource /build/bestsource/ /
# COPY --link --from=akarin /build/akarin/ /
COPY --link --from=vsmlrt /build/vsmlrt/ /

COPY <<EOF /home/ubuntu/.config/vapoursynth/vapoursynth.toml
"/usr/lib/python3/dist-packages/vapoursynth/libvsscript.so" = ["/usr/bin/python3.14","/usr/lib/x86_64-linux-gnu/libpython3.14.so.1.0"]
EOF

VOLUME [ "/models" ]
ENV NVIDIA_DRIVER_CAPABILITIES=all
