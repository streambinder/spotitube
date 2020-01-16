source "zeus/scripts/package-common.sh"

pathSrc="/tmp/spotitube.rpm"
pathPkg="${buildDir}/${binaryName}-v${version}.rpm"

rm -rf "${pathSrc}" ~/rpmbuild
cp -rf "${pathAssets}/rpm" "${pathSrc}"
sed "s|:VERSION:|${version}|g;s|:BINARY:|${buildDir}/${binaryName}|g" \
    "${pathAssets}/rpm/spotitube.spec" > "${pathSrc}/spotitube.spec"
rpmbuild -ba --target=i386 "${pathSrc}/spotitube.spec"
mv ~/rpmbuild/RPMS/i386/*.rpm "${pathPkg}"
rm -rf "${pathSrc}" ~/rpmbuild
