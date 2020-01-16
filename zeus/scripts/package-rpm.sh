source "zeus/scripts/package-common.sh"

src_dir="/tmp/spotitube.rpm"
pkg_dir="${bin_dir}/${bin_name}-v${version}.rpm"

rm -rf "${src_dir}" ~/rpmbuild
cp -rf "${assets_dir}/rpm" "${src_dir}"
sed "s|:VERSION:|${version}|g;s|:BINARY:|${bin_dir}/${bin_name}|g" \
    "${assets_dir}/rpm/spotitube.spec" > "${src_dir}/spotitube.spec"
rpmbuild -ba --target=i386 "${src_dir}/spotitube.spec"
mv ~/rpmbuild/RPMS/i386/*.rpm "${pkg_dir}"
rm -rf "${src_dir}" ~/rpmbuild
