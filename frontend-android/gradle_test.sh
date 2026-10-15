#!/usr/bin/env bash
set -e
export JAVA_HOME=/home/jichi/android-env/jdk-17.0.20+8
export PATH="$JAVA_HOME/bin:$PATH"
export ANDROID_HOME=/home/jichi/.local/android-sdk
export ANDROID_SDK_ROOT="$ANDROID_HOME"

cd /home/jichi/wanli/frontend-android
./gradlew testDebugUnitTest --no-daemon 2>&1 | tail -25
