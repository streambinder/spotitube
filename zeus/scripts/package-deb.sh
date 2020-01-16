source "zeus/scripts/package-common.sh"

src_dir="/tmp/spotitube.deb"
pkg_dir="${bin_dir}/${bin_name}-v${version}.deb"

rm -rf "${src_dir}"
cp -rf "${assets_dir}/deb" "${src_dir}"
sed "s|:VERSION:|${version}|g" \
    "${assets_dir}/deb/DEBIAN/control" > "${src_dir}/DEBIAN/control"
mkdir -p "${src_dir}/usr/sbin"
cp "${bin_dir}/${bin_name}" "${src_dir}/usr/sbin"
dpkg-deb --build "${src_dir}" "${pkg_dir}"
rm -rf "${src_dir}"
