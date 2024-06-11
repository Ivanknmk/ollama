#!/bin/bash

set -e

# Set the source directory
src_dir=$1

if [ -z "$src_dir" ]; then
  echo "Usage: $0 LLAMA_CPP_DIR"
  exit 1
fi

# Set the destination directory
dst_dir=.

# llama.cpp
cp $src_dir/unicode.cpp $dst_dir/unicode.cpp
cp $src_dir/unicode.h $dst_dir/unicode.h
cp $src_dir/unicode-data.cpp $dst_dir/unicode-data.cpp
cp $src_dir/unicode-data.h $dst_dir/unicode-data.h
cp $src_dir/llama.cpp $dst_dir/llama.cpp
cp $src_dir/llama.h $dst_dir/llama.h
cp $src_dir/sgemm.cpp $dst_dir/sgemm.cpp
cp $src_dir/sgemm.h $dst_dir/sgemm.h

# ggml
cp $src_dir/ggml.c $dst_dir/ggml.c
cp $src_dir/ggml.h $dst_dir/ggml.h
cp $src_dir/ggml-quants.c $dst_dir/ggml-quants.c
cp $src_dir/ggml-quants.h $dst_dir/ggml-quants.h
cp $src_dir/ggml-metal.metal $dst_dir/ggml-metal.in.metal
cp $src_dir/ggml-metal.h $dst_dir/ggml-metal.h
cp $src_dir/ggml-metal.m $dst_dir/ggml-metal-darwin_arm64.m
cp $src_dir/ggml-impl.h $dst_dir/ggml-impl.h
cp $src_dir/ggml-cuda.h $dst_dir/ggml-cuda.h
cp $src_dir/ggml-cuda.cu $dst_dir/ggml-cuda.cu
cp $src_dir/ggml-common.h $dst_dir/ggml-common.h
cp $src_dir/ggml-backend.h $dst_dir/ggml-backend.h
cp $src_dir/ggml-backend.c $dst_dir/ggml-backend.c
cp $src_dir/ggml-backend-impl.h $dst_dir/ggml-backend-impl.h
cp $src_dir/ggml-alloc.h $dst_dir/ggml-alloc.h
cp $src_dir/ggml-alloc.c $dst_dir/ggml-alloc.c

# ggml-cuda
mkdir -p $dst_dir/ggml-cuda/template-instances
cp $src_dir/ggml-cuda/*.cu $dst_dir/ggml-cuda/
cp $src_dir/ggml-cuda/*.cuh $dst_dir/ggml-cuda/
cp $src_dir/ggml-cuda/template-instances/*.cu $dst_dir/ggml-cuda/template-instances/

# llava
cp $src_dir/examples/llava/clip.cpp $dst_dir/clip.cpp
cp $src_dir/examples/llava/clip.h $dst_dir/clip.h
cp $src_dir/examples/llava/llava.cpp $dst_dir/llava.cpp
cp $src_dir/examples/llava/llava.h $dst_dir/llava.h
cp $src_dir/common/log.h $dst_dir/log.h
cp $src_dir/common/stb_image.h $dst_dir/stb_image.h

# These files are mostly used by the llava code
# and shouldn't be necessary once we use clip.cpp directly
cp $src_dir/common/common.cpp $dst_dir/common.cpp
cp $src_dir/common/common.h $dst_dir/common.h
cp $src_dir/common/sampling.cpp $dst_dir/sampling.cpp
cp $src_dir/common/sampling.h $dst_dir/sampling.h
cp $src_dir/common/grammar-parser.cpp $dst_dir/grammar-parser.cpp
cp $src_dir/common/grammar-parser.h $dst_dir/grammar-parser.h
cp $src_dir/common/json.hpp $dst_dir/json.hpp
cp $src_dir/common/json-schema-to-grammar.cpp $dst_dir/json-schema-to-grammar.cpp
cp $src_dir/common/json-schema-to-grammar.h $dst_dir/json-schema-to-grammar.h
cp $src_dir/common/base64.hpp $dst_dir/base64.hpp
cat <<EOF > $dst_dir/build-info.cpp
int LLAMA_BUILD_NUMBER = 0;
char const *LLAMA_COMMIT = "$sha1";
char const *LLAMA_COMPILER = "";
char const *LLAMA_BUILD_TARGET = "";
EOF

# apply patches
for patch in $dst_dir/patches/*.diff; do
  git apply "$patch"
done

# add licenses
sha1=$(git -C $src_dir rev-parse @)

TEMP_LICENSE=$(mktemp)
cleanup() {
    rm -f $TEMP_LICENSE
}
trap cleanup 0

cat <<EOF | sed 's/ *$//' >$TEMP_LICENSE
/**
 * llama.cpp - git $sha1
 *
$(sed 's/^/ * /' <$src_dir/LICENSE)
 */

EOF

for IN in $dst_dir/*.{c,h,cpp,m,metal,cu}; do
    if [[ "$IN" == *"sgemm.cpp" || "$IN" == *"sgemm.h" || "$IN" == *"sampling_ext.cpp" || "$IN" == *"sampling_ext.h"  ]]; then
        continue
    fi
    TMP=$(mktemp)
    cat $TEMP_LICENSE $IN >$TMP
    mv $TMP $IN
done
