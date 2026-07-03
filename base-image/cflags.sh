baseflags=(-march=x86-64-v3 -O2 -pipe -fno-plt -fexceptions -ffast-math
    -Wp,-D_FORTIFY_SOURCE=2 -Wformat -Werror=format-security
    -fstack-clash-protection -fcf-protection -flto)
llvm_version=22

export CC=clang-${llvm_version}
export CXX=clang++-${llvm_version}
export CFLAGS="${baseflags[@]}"
export CXXFLAGS="${CFLAGS} -Wp,-D_GLIBCXX_ASSERTIONS"
export LDFLAGS="-Wl,-O1,--sort-common,--as-needed,-z,relro,-z,now,-flto,-fuse-ld=lld-${llvm_version}"
export AR=llvm-ar-${llvm_version}
