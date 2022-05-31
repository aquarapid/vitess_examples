#!/bin/bash
docker run -v `pwd`:/tmp/vitess --net=host -ti --entrypoint /tmp/vitess/launch_standalone_docker.sh vitess/vttestserver:mysql80
