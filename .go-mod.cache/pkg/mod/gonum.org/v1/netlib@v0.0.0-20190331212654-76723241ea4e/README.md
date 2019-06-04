Gonum NETLIB  [![Build Status](https://travis-ci.org/gonum/netlib.svg?branch=master)](https://travis-ci.org/gonum/netlib)  [![codecov.io](https://codecov.io/gh/gonum/netlib/branch/master/graph/badge.svg)](https://codecov.io/gh/gonum/netlib) [![coveralls.io](https://coveralls.io/repos/gonum/netlib/badge.svg?branch=master&service=github)](https://coveralls.io/github/gonum/netlib?branch=master) [![GoDoc](https://godoc.org/gonum.org/v1/netlib?status.svg)](https://godoc.org/gonum.org/v1/netlib)
======

Wrapper packages providing an interface to the NETLIB C BLAS and LAPACKE implementations.

## Installation

```
  go get -d gonum.org/v1/netlib/...
```


Install OpenBLAS:
```
  git clone https://github.com/xianyi/OpenBLAS
  cd OpenBLAS
  make
```

Then install the CGO BLAS wrapper package:
```sh
  CGO_LDFLAGS="-L/path/to/OpenBLAS -lopenblas" go install gonum.org/v1/netlib/blas/netlib
```
or the CGO LAPACKE wrapper package:
```sh
  CGO_LDFLAGS="-L/path/to/OpenBLAS -lopenblas" go install gonum.org/v1/netlib/lapack/netlib
```

For Windows you can download binary packages for OpenBLAS at
http://sourceforge.net/projects/openblas/files/

If you want to use a different BLAS package such as the Intel MKL you can
adjust the `CGO_LDFLAGS` variable:
```sh
  CGO_LDFLAGS="-lmkl_rt" go install gonum.org/v1/netlib/...
```

## Packages

### blas/netlib

Binding to a C implementation of the cblas interface (e.g. ATLAS, OpenBLAS, Intel MKL)

The recommended (free) option for good performance on both Linux and Darwin is OpenBLAS.

### lapack/netlib

Binding to a C implementation of the lapacke interface (e.g. ATLAS, OpenBLAS, Intel MKL)

The recommended (free) option for good performance on both Linux and Darwin is OpenBLAS.

### lapack/lapacke

Low level binding to a C implementation of the lapacke interface (e.g. OpenBLAS or intel MKL)

The linker flags (i.e. path to the BLAS library and library name) might have to be adapted.

The recommended (free) option for good performance on both linux and darwin is OpenBLAS.

## Issues

If you find any bugs, feel free to file an issue on the github issue tracker. Discussions on API changes, added features, code review, or similar requests are preferred on the gonum-dev Google Group.

https://groups.google.com/forum/#!forum/gonum-dev

## License

Please see gonum.org/v1/gonum for general license information, contributors, authors, etc on the Gonum suite of packages.
