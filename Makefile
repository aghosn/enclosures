BOOTFOLDER_NAME='.litter_bootstrap'

# Install the bootstrap
install_bootstrap:
	@-mkdir ~/$(BOOTFOLDER_NAME)
	@tar -xzvf gosb_bootstrap.tar.gz -C ~/$(BOOTFOLDER_NAME)

gosb:
	make -C go/

litterbox:
	make -C LitterBox/
	make -C LitterBox/ install

cpython:
	cd cpython && ./configure
	make -C cython/smalloc-lib/	
	make -C LitterBox/ install




