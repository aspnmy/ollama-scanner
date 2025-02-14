#!/bin/bash
builduser="aspnmy"
buildname="ollama-scanner"
buildver="v2.2"

buildtag_masscan="masscan"
buildtag_zmap="zmap"

builddir_masscan="dockerfile-masscan"
builddir_zmap="dockerfile-zmap"


docker build -f $builddir_masscan -t $builduser/$buildname:$buildver-$buildtag_masscan  .
docker build -f $builddir_zmap -t $builduser/$buildname:$buildver-$buildtag_zmap .