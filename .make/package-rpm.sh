#!/bin/bash

src_dir="/tmp/spotitube.rpm"
pkg_dir="${BUILD_DIR}/${BUILD_NAME}-v${BUILD_VERSION}.rpm"

rm -rf "${src_dir}" ~/rpmbuild
cp -rf "${MAKE_DIR}/package/rpm" "${src_dir}"
sed "s|:VERSION:|${BUILD_VERSION}|g;s|:BINARY:|${BUILD_DIR}/${BUILD_NAME}|g" \
    "${MAKE_DIR}/package/rpm/spotitube.spec" > "${src_dir}/spotitube.spec"
rpmbuild -ba --target=i386 "${src_dir}/spotitube.spec"
mv ~/rpmbuild/RPMS/i386/*.rpm "${pkg_dir}"
rm -rf "${src_dir}" ~/rpmbuild
