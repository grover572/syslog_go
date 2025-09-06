Name:           syslog_go
Version:        1.0.0
Release:        1%{?dist}
Summary:        A syslog client and server tool

License:        MIT
URL:            https://github.com/yourusername/syslog_go
Source0:        %{name}-%{version}.tar.gz

BuildRequires:  golang >= 1.20

%description
A versatile syslog tool that supports both client and server functionality,
with features for sending custom syslog messages and mocking syslog servers.

%prep
%autosetup

%build
go build -o %{name}

%install
rm -rf %{buildroot}
mkdir -p %{buildroot}%{_bindir}
cp -p %{name} %{buildroot}%{_bindir}/

# Install man pages
mkdir -p %{buildroot}%{_mandir}/man1
cp -p doc/man/%{name}.1 %{buildroot}%{_mandir}/man1/

%files
%{_bindir}/%{name}
%{_mandir}/man1/%{name}.1*
%doc README.md
%license LICENSE

%changelog
* Thu Mar 21 2024 Your Name <your.email@example.com> - 1.0.0-1
- Initial package release