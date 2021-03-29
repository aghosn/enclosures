OUT=$(shell pwd)

liblitterbox.so: src/main.go
	GOPATH=$(OUT) gosb build -buildmode=c-shared -o liblitterbox.so src/main.go

install:
	cp liblitterbox.h ../cpython/sandbox-lib/
	cp liblitterbox.so ../cpython/sandbox-lib/libsandbox.so

.PHONY: clean

clean:
	rm -rf liblitterbox.so
