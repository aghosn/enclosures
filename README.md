# Enclosures

## Disclaimer

This repository is a snapshot of a research project that is part of my PhD.
It comes with **NO GUARANTEES** and is **NOT** production-ready.
The `cpython` implementation is a prototype with limited support for python libraries.
As I am finishing my thesis and moving on to new projects, there will be no support in the forseeable future.

The current state of the folder disabled certain features as they are being refactored:

1. Stacks are not protected (we deactivated split-stacks)
2. The MPK backend seccomp support is not included, as it requires a modified kernel and kernel headers, which would make it impossible to compile gosb on unmodified Linux platforms.

## Building

The root folder contains a simple Makefile.

### Building GOSB

The first step requires to install the bootstrap compiler for the go compiler & runtime.
As the bootstrap is a large object, `git lfs` is required to pull the `gosb_bootstrap.tar.gz` file.
Once downloaded, the bootstrap compiler can be installed in the default `$HOME/.litter_bootstrap` folder by running:

```
make install_bootstrap
```

Your environment (e.g., `bashrc`) should define the `GOSB_BOOT` environment variable:

```
export GOSB_BOOT="$HOME/.litter_bootstrap/go/"
```

Now `gosb`, the modified go compiler and runtime, can be compiled by running:

```
make gosb
```

The `go/install.sh` bash script might fail if you do not have the write permissions on `/usr/local/bin`.

### Building LitterBox

The `LitterBox` library requires `gosb` to be available in your `$PATH` environment variable (which should be the case if the previous installation step worked).
It can then be built by running:

```
make LitterBox
```

### Building cpython

Cpython requires configuration to generate the default makefile.
Run the following commands (it might take a while).

```
cd cpython
./configure
cd ..
make cpython
```


