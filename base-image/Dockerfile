#syntax=docker/dockerfile:1-labs

# base image with prepared apt
# it also re-enable apt cache
FROM ubuntu:22.04 as base
ARG ENABLE_APT_CACHE=1
RUN test "${ENABLE_APT_CACHE}" = "1" && rm /etc/apt/apt.conf.d/docker-clean
RUN --mount=target=/var/lib/apt/lists,type=cache,sharing=locked \
	--mount=target=/var/cache/apt,type=cache,sharing=locked \
	apt-get update && apt-get install -y curl gnupg2 && \
	mkdir -p /etc/apt/keyrings/ && \
	curl -L https://repo.radeon.com/rocm/rocm.gpg.key | gpg --dearmor > /etc/apt/keyrings/rocm.gpg && \
	echo "deb [arch=amd64 signed-by=/etc/apt/keyrings/rocm.gpg] https://repo.radeon.com/rocm/apt/6.0.2 jammy main" > /etc/apt/sources.list.d/rocm.list && \
	echo "Package: *\nPin: release o=repo.radeon.com\nPin-Priority: 600" > /etc/apt/preferences.d/rocm-pin-600 && \
	apt-get update
ENV DEBIAN_FRONTEND noninteractive

# ========================================================= #
FROM base AS dev
WORKDIR /workspace
RUN --mount=target=/var/lib/apt/lists,type=cache,sharing=locked \
	--mount=target=/var/cache/apt,type=cache,sharing=locked \
	apt-get install -y gcc-12 g++-12 build-essential git pkg-config

FROM scratch as cflags
COPY <<EOF /cflags.sh
	export CC=/usr/bin/gcc-12
	export CXX=/usr/bin/g++-12
	baseflags=(-march=x86-64-v3 -O2 -pipe -fno-plt -fexceptions
			-Wp,-D_FORTIFY_SOURCE=2 -Wformat -Werror=format-security
			-fstack-clash-protection -fcf-protection -flto)
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
FROM dev AS x265
RUN --mount=target=/var/lib/apt/lists,type=cache,sharing=locked \
	--mount=target=/var/cache/apt,type=cache,sharing=locked \
	apt-get install -y cmake ninja-build

COPY --link --from=cflags /cflags.sh .

ARG X265_VER=3.5
ADD --link https://bitbucket.org/multicoreware/x265_git/get/${X265_VER}.tar.gz x265-${X265_VER}.tar.gz

RUN bash -euxo pipefail <<EOF
	set -euxo pipefail
	tar xf x265-${X265_VER}.tar.gz
	(
		source cflags.sh
		cd multicoreware-x265_git-*
		
		cmake -S source -B build-10 -G Ninja \
			-DCMAKE_INSTALL_PREFIX=/usr \
			-DHIGH_BIT_DEPTH=TRUE \
			-DEXPORT_C_API=FALSE \
			-DENABLE_CLI=FALSE \
			-DENABLE_SHARED=FALSE \
			-Wno-dev
		ninja -C build-10

		cmake -S source -B build -G Ninja \
			-DCMAKE_INSTALL_PREFIX=/usr \
			-DENABLE_SHARED=TRUE \
			-DENABLE_HDR10_PLUS=TRUE \
			-DEXTRA_LIB='x265_main10.a' \
			-DEXTRA_LINK_FLAGS='-L .' \
			-DLINKED_10BIT=TRUE \
			-Wno-dev
			ln -s ../build-10/libx265.a build/libx265_main10.a
			ninja -C build

		DESTDIR="/build/x265" ninja -C build install
	)
	rm -r multicoreware-x265_git-*
EOF

FROM dev AS ffmpeg
RUN --mount=target=/var/lib/apt/lists,type=cache,sharing=locked \
	--mount=target=/var/cache/apt,type=cache,sharing=locked \
	apt-get install -y nasm libtheora-dev libopus-dev libvpx-dev libx264-dev libzimg-dev python3-dev

COPY --link --from=cflags /cflags.sh .

ARG FFMPEG_VER=6.1.1
ADD --link https://ffmpeg.org/releases/ffmpeg-${FFMPEG_VER}.tar.xz .

ARG AMF_VER=1.4.33
ADD --link https://github.com/abihf/AMF/releases/download/v${AMF_VER}/AMF-headers.tar.gz AMF-headers.tar.gz
ADD --link https://patchwork.ffmpeg.org/project/ffmpeg/patch/20231023134753.17106-3-lucenticus@gmail.com/raw/ amf-10bit.patch

COPY --link --from=x265 /build/x265/ /

RUN bash -euxo pipefail <<EOF
	tar xf AMF-${AMF_VER}.tar.gz -C /usr/include
	tar xf ffmpeg-${FFMPEG_VER}.tar.xz
	(
		source cflags.sh
		cd ffmpeg-${FFMPEG_VER}
		patch -p1 < ../amf-10bit.patch
		./configure \
			--prefix=/usr \
			--disable-debug \
			--disable-static \
			--enable-amf \
			--enable-gpl \
			--enable-lto \
			--enable-libopus \
			--enable-libtheora \
			--enable-libvpx \
			--enable-libx264 \
			--enable-libx265 \
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

FROM dev AS rocm-dev
RUN --mount=target=/var/lib/apt/lists,type=cache,sharing=locked \
	--mount=target=/var/cache/apt,type=cache,sharing=locked \
	apt-get install -y --no-install-recommends \
	hip-runtime-amd rocm-device-libs rocm-dev miopen-hip-dev rocblas-dev half

FROM rocm-dev AS migraphx
RUN --mount=target=/var/lib/apt/lists,type=cache,sharing=locked \
	--mount=target=/var/cache/apt,type=cache,sharing=locked \
	apt-get install -y  --no-install-recommends \
	cmake ninja-build python3-pip \
	hip-base \
	libnuma-dev \
	hipfft \
	rocthrust \
	rocrand \
	hipsparse \
	rccl \
	rccl-dev \
	rocm-smi-lib \
	roctracer-dev \
	hipcub  \
	hipblas  \
	hipify-clang \
	hiprand-dev \
	libssl-dev \
	zlib1g-dev

RUN pip3 install https://github.com/RadeonOpenCompute/rbuild/archive/master.tar.gz

ARG MIGRAPHX_VER=84fc9f0207d37f337eb7696f5981ac5275d1b1ff
ADD --link https://github.com/ROCm/AMDMIGraphX.git#${MIGRAPHX_VER} AMDMIGraphX

RUN cd AMDMIGraphX && rbuild prepare -d depend

RUN bash -euxo pipefail <<EOF	(
		cd AMDMIGraphX
		rbuild package -d depend -B build -DGPU_TARGETS=gfx1030 -DCMAKE_BUILD_TYPE=Release
		mv build/*.deb .
		rm -rf build
	)
EOF

# ========================================================= #

FROM rocm-dev AS vsmlrt
RUN --mount=target=/var/lib/apt/lists,type=cache,sharing=locked \
	--mount=target=/var/cache/apt,type=cache,sharing=locked \
	apt-get update && \
	apt-get install -y  --no-install-recommends \
	cmake ninja-build

COPY --link --from=cflags /cflags.sh .

ARG VSMLRT_VER=e775824473210c05009227c897a85e05fe748a25
ADD --link --keep-git-dir=true https://github.com/abihf/vs-mlrt.git#${VSMLRT_VER} vs-mlrt

COPY --link --from=vapoursynth /build/vapoursynth/ /

RUN --mount=target=/workspace/AMDMIGraphX,source=/workspace/AMDMIGraphX,type=bind,from=migraphx \
	cd /workspace/AMDMIGraphX && dpkg -i migraphx_2.10.0-_amd64.deb migraphx-dev_2.10.0_amd64.deb

RUN bash <<EOF
	set -euxo pipefail
	(
		source cflags.sh
		cd vs-mlrt/vsmigx
		cmake cmake -S . -B build -G Ninja -LA \
			-D VAPOURSYNTH_INCLUDE_DIRECTORY=/usr/include/vapoursynth \
			-D CMAKE_BUILD_TYPE=Debug \
			-D CMAKE_CXX_COMPILER=g++-12 \
			-D CMAKE_CXX_FLAGS="-Wall -ffast-math -march=x86-64-v3" \
			-D migraphx_DIR=/opt/rocm/lib/cmake/migraphx \
			-D MIOpen_DIR=/opt/rocm/lib/cmake/miopen \
			-D hip_DIR=/opt/rocm/lib/cmake/hip \
			-D AMDDeviceLibs_DIR=/opt/rocm/lib/cmake/AMDDeviceLibs \
			-D amd_comgr_DIR=/opt/rocm/lib/cmake/amd_comgr \
			-D hsa-runtime64_DIR=/opt/rocm/lib/cmake/hsa-runtime64 \
			-D rocblas_DIR=/opt/rocm/lib/cmake/rocblas
		cmake --build build
	)
	mkdir -p /build/vsmlrt/usr/lib/vapoursynth/vsmlrt-hip/
	mkdir -p /build/vsmlrt/usr/lib/python3/dist-packages/
	cp vs-mlrt/vsmigx/build/libvsmigx.so /build/vsmlrt/usr/lib/vapoursynth
	cp vs-mlrt/scripts/vsmlrt.py /build/vsmlrt/usr/lib/python3/dist-packages/
	ln -sf /models /build/vsmlrt/usr/lib/vapoursynth/models
	ln -s /opt/rocm/bin/migraphx-driver /build/vsmlrt/usr/lib/vapoursynth/vsmlrt-hip/
	ln -sf /opt/rocm/lib/libmigraphx_gpu.so.2010000.0 /build/vsmlrt/usr/lib/libmigraphx_gpu.so
EOF

# ========================================================= #

FROM base AS amdgpu-pro
WORKDIR /amdgpu-pro
ADD --link https://repo.radeon.com/amdgpu/6.0.2/ubuntu/pool/proprietary/a/amdgpu-pro-core/amdgpu-pro-core_23.40-1718239.20.04_all.deb .
ADD --link https://repo.radeon.com/amdgpu/6.0.2/ubuntu/pool/proprietary/a/amf-amdgpu-pro/amf-amdgpu-pro_1.4.33-1718239.20.04_amd64.deb .
ADD --link https://repo.radeon.com/amdgpu/6.0.2/ubuntu/pool/proprietary/liba/libamdenc-amdgpu-pro/libamdenc-amdgpu-pro_1.0-1718239.20.04_amd64.deb .
ADD --link https://repo.radeon.com/amdgpu/6.0.2/ubuntu/pool/main/libd/libdrm-amdgpu/libdrm2-amdgpu_2.4.116.60002-1718217.22.04_amd64.deb .
ADD --link https://repo.radeon.com/amdgpu/6.0.2/ubuntu/pool/main/a/amdgpu-core/amdgpu-core_6.0.60002-1718217.22.04_all.deb .

FROM base AS runtime
RUN --mount=target=/var/lib/apt/lists,type=cache,sharing=locked \
	--mount=target=/var/cache/apt,type=cache,sharing=locked \
	apt-get install -y --no-install-recommends \
	libpython3.10 hip-runtime-amd rocm-device-libs miopen-hip rocblas roctracer half \
	libtheora0 libzimg2 libopus0 libvpx7 libx264-163 libx265-199 

RUN --mount=target=/workspace/AMDMIGraphX,source=/workspace/AMDMIGraphX,type=bind,from=migraphx \
	cd /workspace/AMDMIGraphX/build && dpkg -i migraphx_2.10.0-_amd64.deb

RUN --mount=target=/var/lib/apt/lists,type=cache,sharing=locked \
	--mount=target=/var/cache/apt,type=cache,sharing=locked \
	--mount=target=/amdgpu-pro,source=/amdgpu-pro,type=bind,from=amdgpu-pro \
	cd /amdgpu-pro && (dpkg -i *.deb || apt install -y --fix-broken)

COPY --link --from=x265 /build/x265/ /
COPY --link --from=ffmpeg /build/ffmpeg/ /
COPY --link --from=vapoursynth /build/vapoursynth/ /
COPY --link --from=lsmash /build/lsmash/ /
COPY --link --from=vsmlrt /build/vsmlrt/ /
VOLUME [ "/models" ]
ENV NVIDIA_DRIVER_CAPABILITIES=all
