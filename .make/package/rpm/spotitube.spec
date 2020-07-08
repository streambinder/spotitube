Name:      spotitube
Version:   :VERSION:
Release:   :VERSION:%{?dist}
Summary:   Synchronize your Spotify collections downloading from external providers.
Group:     SpotiTube
License:   GPL
BuildRoot: %(mktemp -ud %{_tmppath}/%{name}-%{version}-%{release}-XXXXXX)
Requires:  ffmpeg, youtube-dl

%description
Synchronize your Spotify collections downloading from external providers.

%prep
exit 0

%build
exit 0

%install
install --directory $RPM_BUILD_ROOT/usr/sbin
install -m 0755 :BINARY: $RPM_BUILD_ROOT/usr/sbin

%clean
rm -rf $RPM_BUILD_ROOT

%files
/usr/sbin/spotitube

%changelog
