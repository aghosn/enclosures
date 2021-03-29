TARGET=main.out
CC=gosb
OUT=$(shell pwd)
LITTER?=VTX
BENCH?=
ARG1?=10000
ARG2?=10000

all: $(TARGET)

$(TARGET): src/main.go
	$(CC) build -a -o $@ $^

.SILENT:
benchmark: $(TARGET)
	GOMAXPROCS=1 LITTER=$(LITTER) BENCH=$(BENCH) ARG1=$(ARG1) ARG2=$(ARG2) ./main.out

clean: 
	rm -f $(TARGET)


