#!/bin/bash

src_dir="/tmp/spotitube.eopkg"
pkg_dir="${BUILD_DIR}/${BUILD_NAME}-v${BUILD_VERSION}.eopkg"

rm -rf "${src_dir}"
sudo solbuild update
cp -rf "${MAKE_DIR}/package/eopkg" "${src_dir}"
sed "s|:VERSION:|${BUILD_VERSION}|g" \
    "${MAKE_DIR}/package/eopkg/pspec.xml" > "${src_dir}/pspec.xml"
cp "${BUILD_DIR}/${BUILD_NAME}" "${src_dir}/files"
sudo solbuild build "${src_dir}/pspec.xml"
mv spotitube-*.eopkg "${pkg_dir}"
rm -rf "${src_dir}"
