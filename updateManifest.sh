#!/bin/bash -v

docker manifest create dengrenjie31/rank \
	dengrenjie31/rank:amd64 \
	dengrenjie31/rank:aarch64

docker manifest annotate dengrenjie31/rank:amd64 \
	--os linux --arch amd64

docker manifest annotate dengrenjie31/rank:aarch64 \
	--os linux --arch arm64 

docker manifest push dengrenjie31/rank

docker manifest rm dengrenjie31/rank

