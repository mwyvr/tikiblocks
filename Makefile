# tikiblocks, a fork of goblocks
VERSION = 0.1

PREFIX = /usr
MANPREFIX = $(PREFIX)/share/man

build:
	go build tikiblocks.go

config:
	./checkConfig

install: build
	mkdir -p ${DESTDIR}${PREFIX}/bin
	cp -f tikiblocks ${DESTDIR}${PREFIX}/bin
	chmod 755 ${DESTDIR}${PREFIX}/bin/tikiblocks
	mkdir -p ${DESTDIR}${MANPREFIX}/man1
	sed "s/VERSION/${VERSION}/g" < tikiblocks.1 > ${DESTDIR}${MANPREFIX}/man1/tikiblocks.1
	chmod 644 ${DESTDIR}${MANPREFIX}/man1/tikiblocks.1

uninstall:
	sudo rm -f ${DESTDIR}${PREFIX}/bin/tikiblocks \
		${DESTDIR}${PREFIX}/man1/tikiblocks.1 \
	# in case installed directly via Go
	rm -f ${HOME}/go/bin/tikiblocks

run: build
	./tikiblocks -o xprop

