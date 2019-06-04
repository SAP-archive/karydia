#!/bin/bash
set -ex

go generate gonum.org/v1/netlib/blas/netlib
go generate gonum.org/v1/netlib/lapack/lapacke
go generate gonum.org/v1/netlib/lapack/netlib

git checkout -- go.{mod,sum}
if [ -n "$(git diff)" ]; then
	git diff
	exit 1
fi
