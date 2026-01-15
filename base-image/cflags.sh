baseflags=(-march=x86-64-v3 -O2 -pipe -fno-plt -fexceptions -ffast-math
    -Wp,-D_FORTIFY_SOURCE=2 -Wformat -Werror=format-security
    -fstack-clash-protection -fcf-protection -flto)
export CC=clang-21
export CXX=clang++-21
export CFLAGS="${baseflags[@]}"
export CXXFLAGS="${CFLAGS} -Wp,-D_GLIBCXX_ASSERTIONS"
export LDFLAGS="-Wl,-O1,--sort-common,--as-needed,-z,relro,-z,now,-flto,-fuse-ld=lld-21"
export AR=llvm-ar-21