
.SILENT:
compile:
	@make -C Call -B
	@make -C Register -B
	@make -C Syscall -B
	@echo "Done compiling"

benchmark:
	@make -C Call benchmark
	@make -C Register benchmark
	@make -C Syscall benchmark

.PHONY: clean

clean:
	@make -C Call clean 
	@make -C Register clean
	@make -C Syscall clean
	@echo "Done cleaning"
