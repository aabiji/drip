#!/bin/sh

mkdir -p build && cd build
gogio -ldflags "-checklinkname=0" -icon ../assets/logo.png -target android github.com/aabiji/drip &&
adb install drip.apk
