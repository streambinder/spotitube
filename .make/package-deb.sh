#!/bin/bash

src_dir="/tmp/spotitube.deb"
pkg_dir="${BUILD_DIR}/${BUILD_NAME}-v${BUILD_VERSION}.deb"

rm -rf "${src_dir}"
cp -rf "${MAKE_DIR}/package/deb" "${src_dir}"
sed "s|:VERSION:|${BUILD_VERSION}|g" \
    "${MAKE_DIR}/package/deb/DEBIAN/control" > "${src_dir}/DEBIAN/control"
mkdir -p "${src_dir}/usr/sbin"
cp "${BUILD_DIR}/${BUILD_NAME}" "${src_dir}/usr/sbin"
dpkg-deb --build "${src_dir}" "${pkg_dir}"
rm -rf "${src_dir}"
