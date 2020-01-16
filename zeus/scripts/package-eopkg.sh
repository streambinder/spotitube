source "zeus/scripts/package-common.sh"

src_dir="/tmp/spotitube.eopkg"
pkg_dir="${bin_dir}/${bin_name}-v${version}.eopkg"

rm -rf "${src_dir}"
sudo solbuild update
cp -rf "${assets_dir}/eopkg" "${src_dir}"
sed "s|:VERSION:|${version}|g" \
    "${assets_dir}/eopkg/pspec.xml" > "${src_dir}/pspec.xml"
cp "${bin_dir}/${bin_name}" "${src_dir}/files"
sudo solbuild build "${src_dir}/pspec.xml"
mv spotitube-*.eopkg "${pkg_dir}"
rm -rf "${src_dir}"
