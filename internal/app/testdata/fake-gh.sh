#!/bin/sh
# Answers `gh pr view` with the state in $FAKE_GH_STATE (MERGED, CLOSED or OPEN).
echo "{\"state\":\"$FAKE_GH_STATE\",\"mergedAt\":null,\"closedAt\":null,\"labels\":[],\"isDraft\":false,\"mergeable\":\"MERGEABLE\"}"
