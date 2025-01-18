#syntax=docker/dockerfile:1-labs


# base image with prepared apt
# it also re-enable apt cache
FROM ubuntu:22.04 as base
ARG ENABLE_APT_CACHE=1
RUN test "${ENABLE_APT_CACHE}" = "1" && rm /etc/apt/apt.conf.d/docker-clean
ENV DEBIAN_FRONTEND noninteractive
RUN --mount=target=/var/cache/apt,type=cache,sharing=locked bash -euxo pipefail <<EOF
	sed -i "s|deb [a-z]*://[^ ]* |deb http://linux.domainesia.com/ubuntu/ubuntu-archive/ |g" /etc/apt/sources.list
	apt-get update && apt-get install -y curl gnupg2
	mkdir -p /etc/apt/keyrings/
	curl -L https://repo.radeon.com/rocm/rocm.gpg.key | gpg --dearmor > /etc/apt/keyrings/rocm.gpg
	echo "deb [arch=amd64 signed-by=/etc/apt/keyrings/rocm.gpg] https://repo.radeon.com/rocm/apt/6.2.2 jammy main" > /etc/apt/sources.list.d/rocm.list
	echo "deb [arch=amd64 signed-by=/etc/apt/keyrings/rocm.gpg] https://repo.radeon.com/rocm/apt/6.2.2 jammy proprietary" >> /etc/apt/sources.list.d/rocm.list
	echo "deb [arch=amd64 signed-by=/etc/apt/keyrings/rocm.gpg] https://repo.radeon.com/amdgpu/6.2.2/ubuntu jammy main" > /etc/apt/sources.list.d/amdgpu.list
	echo "deb [arch=amd64 signed-by=/etc/apt/keyrings/rocm.gpg] https://repo.radeon.com/amdgpu/6.2.2/ubuntu jammy proprietary" >> /etc/apt/sources.list.d/amdgpu.list
	echo -e 'Package: *\nPin: release o=repo.radeon.com\nPin-Priority: 600' > /etc/apt/preferences.d/rocm-pin-600
	apt-get update
	apt-get -y upgrade
EOF

# ========================================================= #
FROM base AS dev
WORKDIR /workspace
RUN --mount=target=/var/cache/apt,type=cache,sharing=locked \
	apt-get install -y gcc-12 g++-12 build-essential git pkg-config

FROM scratch as cflags
COPY <<EOF /cflags.sh
	export CC=/usr/bin/gcc-12
	export CXX=/usr/bin/g++-12
	baseflags=(-march=x86-64-v4 -O2 -pipe -fno-plt -fexceptions
			-Wp,-D_FORTIFY_SOURCE=2 -Wformat -Werror=format-security
			-fstack-clash-protection -fcf-protection -flto)
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

FROM dev AS ffmpeg
RUN --mount=target=/var/cache/apt,type=cache,sharing=locked \
	apt-get install -y nasm libtheora-dev libopus-dev libvpx-dev libx264-dev libx265-dev libzimg-dev libva-dev

COPY --link --from=cflags /cflags.sh .

ARG FFMPEG_VER=n7.1
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
			--enable-vaapi \
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
	apt-get install -y meson libzimg-dev llvm-dev zlib1g-dev

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
	apt-get install -y  --no-install-recommends \
	cmake ninja-build migraphx-dev miopen-hip-dev rocblas-dev hipcc rocm-device-libs

COPY --link --from=cflags /cflags.sh .

ARG VSMLRT_VER=935728d722b110e8acd423974d66b0dbb7a1b17e
ADD --link --keep-git-dir=true https://github.com/AmusementClub/vs-mlrt.git#${VSMLRT_VER} vs-mlrt

COPY --link --from=vapoursynth /build/vapoursynth/ /

RUN bash <<EOF
	set -euxo pipefail
	(
		source cflags.sh
		cd vs-mlrt/vsmigx
		sed -i /version.h/d vs_migraphx.cpp 
		sed -i 's/MIGRAPHX_VERSION_/HIP_VERSION_/g' vs_migraphx.cpp
		cmake cmake -S . -B build -G Ninja -LA \
			-D VAPOURSYNTH_INCLUDE_DIRECTORY=/usr/include/vapoursynth \
			-D CMAKE_BUILD_TYPE=Release \
			-D CMAKE_CXX_COMPILER=g++-12 \
			-D CMAKE_CXX_FLAGS="-Wall -ffast-math -march=x86-64-v4 -DHIP_VERSION_TWEAK  -DMIGRAPHX_VERSION_TWEAK" \
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
	libpython3.10 half libnuma1 \
	libtheora0 libzimg2 libopus0 libvpx7 libx264-163 libx265-199 libva2 libva-drm2 mesa-va-drivers

COPY --link --from=ffmpeg /build/ffmpeg/ /
COPY --link --from=vapoursynth /build/vapoursynth/ /
COPY --link --from=lsmash /build/lsmash/ /
COPY --link --from=akarin /build/akarin/ /
COPY --link --from=vsmlrt /build/vsmlrt/ /

RUN echo "/opt/rocm/lib" > /etc/ld.so.conf.d/rocm.conf && ldconfig

VOLUME [ "/models" ]
