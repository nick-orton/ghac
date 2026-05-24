.PHONY: build test test-integration clean install

PREFIX?=	/usr/local
BINDIR?=	${PREFIX}/bin

build:
	go build -buildvcs=false -o ghac ./cmd/ghac

test:
	go test ./...

test-integration:
	go test -tags integration ./...

clean:
	rm -f ghac

install: build
	install -m 555 ghac ${DESTDIR}${BINDIR}/ghac
