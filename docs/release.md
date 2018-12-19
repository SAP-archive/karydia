karydia doesn't have a release schedule. When we deem a new release is
needed or due, we tag the tip of master.

## Example workflow

```
git checkout master
git pull
```

Make sure you are on the right commit and everything looks good.

Create an annotated tag:

```
git tag -a -m 0.1.0 0.1.0
git push origin 0.1.0
```

Build a new container:

```
make container

[...]
Successfully built 2c5771a01673
Successfully tagged karydia/karydia:latest
```

Tag it with the right release (here tag build `2c5771a01673` as `0.1.0`):

```
docker tag 2c5771a01673 karydia/karydia:0.1.0
```

Push both images to Docker Hub:

```
docker push karydia/karydia:latest
[...]
docker push karydia/karydia:0.1.0
[...]
```

Update manifests and documentation where necessary.

That's it!
