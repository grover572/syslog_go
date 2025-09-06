# 编译和安装目标
.PHONY: all install install-man clean

PREFIX ?= /usr/local
MANPREFIX ?= $(PREFIX)/share/man

all: syslog_go

syslog_go:
	go build

install: syslog_go install-man
	@mkdir -p $(PREFIX)/bin
	@cp syslog_go $(PREFIX)/bin/
	@chmod 755 $(PREFIX)/bin/syslog_go
	@echo "已安装 syslog_go 到 $(PREFIX)/bin"

install-man:
	@mkdir -p $(MANPREFIX)/man1
	@cp doc/man/syslog_go.1 $(MANPREFIX)/man1/
	@chmod 644 $(MANPREFIX)/man1/syslog_go.1
	@echo "已安装 man 手册到 $(MANPREFIX)/man1"

rpm:
	chmod +x scripts/build_rpm.sh
	./scripts/build_rpm.sh

test-build:
	chmod +x scripts/test_build.sh
	./scripts/test_build.sh

clean:
	@rm -f syslog_go