source "zeus/scripts/package-common.sh"

pathSrc="/tmp/spotitube.deb"
pathPkg="${buildDir}/${binaryName}-v${version}.deb"

rm -rf "${pathSrc}"
cp -rf "${pathAssets}/deb" "${pathSrc}"
sed "s|:VERSION:|${version}|g" \
    "${pathAssets}/deb/DEBIAN/control" > "${pathSrc}/DEBIAN/control"
mkdir -p "${pathSrc}/usr/sbin"
cp "${buildDir}/${binaryName}" "${pathSrc}/usr/sbin"
dpkg-deb --build "${pathSrc}" "${pathPkg}"
rm -rf "${pathSrc}"
