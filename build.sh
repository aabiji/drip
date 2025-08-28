#!/bin/sh

javac -classpath $ANDROID_HOME/platforms/android-36/android.jar -d /tmp/java_classes android_utility.java
jar cf android_utility.jar -C /tmp/java_classes .

mkdir -p build && cd build
gogio -ldflags "-checklinkname=0" -icon ../assets/logo.png -target android github.com/aabiji/drip
adb install drip.apk
