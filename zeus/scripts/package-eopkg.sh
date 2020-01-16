source "zeus/scripts/package-common.sh"

pathSrc="/tmp/spotitube.eopkg"
pathPkg="${buildDir}/${binaryName}-v${version}.eopkg"

rm -rf "${pathSrc}"
sudo solbuild update
cp -rf "${pathAssets}/eopkg" "${pathSrc}"
sed "s|:VERSION:|${version}|g" \
    "${pathAssets}/eopkg/pspec.xml" > "${pathSrc}/pspec.xml"
cp "${buildDir}/${binaryName}" "${pathSrc}/files"
sudo solbuild build "${pathSrc}/pspec.xml"
mv spotitube-*.eopkg "${pathPkg}"
rm -rf "${pathSrc}"
