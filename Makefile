.PHONY: build
build:
	CGOFLAGS="-buildvcs=false" CGO_ENABLED=0 GOOS=linux go build -ldflags "-s -w" -o restree

.PHONY: clean
clean:
	rm -rf restree

.PHONY: install
install: build
	mkdir -p ${DESTDIR}${PREFIX}/bin
	cp -f restree ${DESTDIR}${PREFIX}/bin
	chmod 755 ${DESTDIR}${PREFIX}/bin/restree

.PHONY: install
uninstall:
	rm -f ${DESTDIR}${PREFIX}/bin/restree

.PHONY: lint
lint:
	golangci-lint run

.PHONY: lint-fix
lint-fix:
	golangci-lint run

.PHONY: format
format:
	golangci-lint fmt

.PHONY: format-check
format-check:
	golangci-lint fmt -d
