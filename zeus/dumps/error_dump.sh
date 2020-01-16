#!/bin/bash
#
# ZEUS Error Dump
# Timestamp: [Thu Jan 16 14:47:09 2020]
# Error: exit status 1
# StdErr: 
# error: cannot open Packages database in /var/lib/rpm
# + umask 022
# + cd /home/streambinder/rpmbuild/BUILD
# + exit 0
# + umask 022
# + cd /home/streambinder/rpmbuild/BUILD
# + exit 0
# + umask 022
# + cd /home/streambinder/rpmbuild/BUILD
# + install --directory /home/streambinder/rpmbuild/BUILDROOT/spotitube-25-25.i386/usr/sbin
# + install -m 0755 /home/streambinder/Workspace/spotitube/bin/spotitube /home/streambinder/rpmbuild/BUILDROOT/spotitube-25-25.i386/usr/sbin
# + /usr/lib/rpm/brp-compress
# + /usr/lib/rpm/brp-strip /usr/bin/strip
# + /usr/lib/rpm/brp-strip-static-archive /usr/bin/strip
# + /usr/lib/rpm/brp-strip-comment-note /usr/bin/strip /usr/bin/objdump
# warning: Missing build-id in /home/streambinder/rpmbuild/BUILDROOT/spotitube-25-25.i386/usr/sbin/spotitube
# error: Installed (but unpackaged) file(s) found:
#    /usr/sbin/spotitube-v25
#     cannot open Packages database in /var/lib/rpm
#     Missing build-id in /home/streambinder/rpmbuild/BUILDROOT/spotitube-25-25.i386/usr/sbin/spotitube
#     Installed (but unpackaged) file(s) found:
#    /usr/sbin/spotitube-v25
# 


#!/bin/bash
binaryName="spotitube"
buildDir="bin"
ldflags="-s -w"
version=25



source "zeus/scripts/package-common.sh"

pathSrc="/tmp/spotitube.rpm"
pathPkg="${buildDir}/${binaryName}-v${version}.rpm"

rm -rf "${pathSrc}"
cp -rf "${pathAssets}/rpm" "${pathSrc}"
sed "s|:VERSION:|${version}|g;s|:BINARY:|${buildDir}/${binaryName}|g" \
    "${pathAssets}/rpm/spotitube.spec" > "${pathSrc}/spotitube.spec"
rpmbuild -ba --target=i386 "${pathSrc}/spotitube.spec"
mv ~/rpmbuild/RPMS/i386/*.rpm "${pathPkg}"
rm -rf ~/rpmbuild
rm -f "${pathSrc}"
