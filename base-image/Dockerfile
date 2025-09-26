# base image with prepared apt
# it also re-enable apt cache
FROM ubuntu:24.04 AS base
ARG ENABLE_APT_CACHE=1
RUN test "${ENABLE_APT_CACHE}" = "1" && rm /etc/apt/apt.conf.d/docker-clean
ENV DEBIAN_FRONTEND noninteractive
RUN --mount=target=/var/cache/apt,type=cache,sharing=locked bash -euxo pipefail <<EOF
	apt-get update && apt-get install -y curl gnupg2 rsync
	curl -fsSL https://repo.radeon.com/amdgpu-install/7.0.1/ubuntu/noble/amdgpu-install_7.0.1.70001-1_all.deb -o /tmp/amdgpu-install.deb
	dpkg -i /tmp/amdgpu-install.deb
	rm /tmp/amdgpu-install.deb
	apt-get update
	apt-get -y upgrade
EOF


# ========================================================= #
FROM base AS dev
WORKDIR /workspace
RUN --mount=target=/var/cache/apt,type=cache,sharing=locked \
	apt-get install -y build-essential git pkg-config gpg
COPY <<EOF /etc/apt/sources.list.d/llvm.list
deb https://apt.llvm.org/noble/ llvm-toolchain-noble-21 main
EOF
RUN --mount=target=/var/cache/apt,type=cache,sharing=locked \
	curl -fsSL https://apt.llvm.org/llvm-snapshot.gpg.key | apt-key add - && \
	apt-get update && \
	apt-get install -y clang-21 clang++-21 lld-21

FROM scratch AS cflags
COPY <<EOF /cflags.sh
	baseflags=(-march=x86-64-v3 -O2 -pipe -fno-plt -fexceptions -ffast-math
			-Wp,-D_FORTIFY_SOURCE=2 -Wformat -Werror=format-security
			-fstack-clash-protection -fcf-protection -flto)
	export CC=clang-21
	export CXX=clang++-21
	export CFLAGS="\${baseflags[@]}"
	export CXXFLAGS="\${CFLAGS} -Wp,-D_GLIBCXX_ASSERTIONS"
	export LDFLAGS="-Wl,-O1,--sort-common,--as-needed,-z,relro,-z,now,-flto,-fuse-ld=lld-21"
	export AR=llvm-ar-21
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
	apt-get install -y nasm libtheora-dev libopus-dev libvpx-dev libzimg-dev python3-dev libva-dev

COPY --link --from=cflags /cflags.sh .

ARG FFMPEG_VER=n8.0
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
			--extra-ldflags="-fuse-ld=lld-21" \
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
			--enable-vaapi \
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
ARG BESTSOURCE_VER=R13
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
	apt-get install -y meson libzimg-dev llvm-21-dev zlib1g-dev libzstd-dev

COPY --link --from=cflags /cflags.sh .

ARG AKARINVS_VER=1.0.0
ADD --link https://github.com/Jaded-Encoding-Thaumaturgy/akarin-vapoursynth-plugin/archive/v${AKARINVS_VER}.tar.gz vapoursynth-plugin-v${AKARINVS_VER}.tar.gz

COPY --link --from=vapoursynth /build/vapoursynth/ /
RUN bash -euxo pipefail <<EOF
	tar xf vapoursynth-plugin-v${AKARINVS_VER}.tar.gz
	(
		source cflags.sh
		cd akarin-vapoursynth-plugin-${AKARINVS_VER}
		export LLVM_CONFIG=llvm-config-21
		meson setup --prefix=/usr build -Dstatic-llvm=true
		meson compile -C build
		meson install --destdir /build/akarin -C build
	)
	rm -rf akarin-vapoursynth-plugin-${AKARINVS_VER}
EOF

# ========================================================= #

FROM dev AS vsmlrt
RUN --mount=target=/var/cache/apt,type=cache,sharing=locked \
	apt-get install -y  --no-install-recommends \
	cmake ninja-build migraphx-dev miopen-hip-dev rocblas-dev hipcc rocm-device-libs hipblaslt-dev

COPY --link --from=cflags /cflags.sh .

ARG VSMLRT_VER=bd48cc73430fb7826456415093c5dacd0f39c125
ADD --link --keep-git-dir=true https://github.com/AmusementClub/vs-mlrt.git#${VSMLRT_VER} vs-mlrt

COPY --link --from=vapoursynth /build/vapoursynth/ /

RUN bash <<EOF
	set -euxo pipefail
	(
		source cflags.sh
		cd vs-mlrt/vsmigx
		# sed -i /version.h/d vs_migraphx.cpp 
		# sed -i 's/MIGRAPHX_VERSION_/HIP_VERSION_/g' vs_migraphx.cpp
		cmake cmake -S . -B build -G Ninja -LA \
			-D VAPOURSYNTH_INCLUDE_DIRECTORY=/usr/include/vapoursynth \
			-D CMAKE_BUILD_TYPE=Release \
			-D CMAKE_CXX_COMPILER=\$CXX \
			-D CMAKE_CXX_FLAGS="-Wall -ffast-math -march=x86-64-v4" \
			-D migraphx_DIR=/opt/rocm/lib/cmake/migraphx \
			-D MIOpen_DIR=/opt/rocm/lib/cmake/miopen \
			-D hip_DIR=/opt/rocm/lib/cmake/hip \
			-D AMDDeviceLibs_DIR=/opt/rocm/lib/cmake/AMDDeviceLibs \
			-D amd_comgr_DIR=/opt/rocm/lib/cmake/amd_comgr \
			-D hsa-runtime64_DIR=/opt/rocm/lib/cmake/hsa-runtime64 \
			-D rocblas_DIR=/opt/rocm/lib/cmake/rocblas \
			-D hipblaslt_DIR=/opt/rocm/lib/cmake/hipblaslt \
			-D CMAKE_PREFIX_PATH=/opt/rocm/lib/cmake
		cmake --build build
	)
	mkdir -p /build/vsmlrt/usr/lib/vapoursynth/vsmlrt-hip/
	mkdir -p /build/vsmlrt/usr/lib/python3/dist-packages/
	cp vs-mlrt/vsmigx/build/libvsmigx.so /build/vsmlrt/usr/lib/vapoursynth
	cp vs-mlrt/scripts/vsmlrt.py /build/vsmlrt/usr/lib/python3/dist-packages/
	ln -sf /models /build/vsmlrt/usr/lib/vapoursynth/models
	ln -s /opt/rocm/bin/migraphx-driver /build/vsmlrt/usr/lib/vapoursynth/vsmlrt-hip/
EOF

# remove unused rocblas & llvm: save about 7.5GB
RUN find /opt/rocm/lib/rocblas/library/ -type f -not -name TensileManifest.txt -not -name '*gfx1100*' -delete && \ 
	rm -rf /opt/rocm/lib/llvm/ && \
	mkdir /build/vsmlrt/opt && \
	mv /opt/rocm* /build/vsmlrt/opt/
# RUN mkdir /build/vsmlrt/opt && \
# 	mv /opt/rocm* /build/vsmlrt/opt/

# ========================================================= #

FROM base AS runtime
RUN sleep 3
RUN --mount=target=/var/cache/apt,type=cache,sharing=locked \
	apt-get install -y --no-install-recommends \
	libpython3.12t64 half libnuma1 \
	libtheora0 libzimg2 libopus0 libvpx9 libva2 mesa-amdgpu-va-drivers libelf1t64 libva-amdgpu-drm2

COPY --link --from=ffmpeg /build/ffmpeg/ /
COPY --link --from=vapoursynth /build/vapoursynth/ /
COPY --link --from=bestsource /build/bestsource/ /
#COPY --link --from=akarin /build/akarin/ /
COPY --link --from=vsmlrt /build/vsmlrt/ /

RUN echo "/opt/rocm/lib" > /etc/ld.so.conf.d/rocm.conf && ldconfig

VOLUME [ "/models" ]
