#!/usr/bin/env bash
set -e
export JAVA_HOME=/home/jichi/android-env/jdk-17.0.20+8
export PATH="$JAVA_HOME/bin:$PATH"
export ANDROID_HOME=/home/jichi/.local/android-sdk
export ANDROID_SDK_ROOT="$ANDROID_HOME"

cd /home/jichi/wanli/frontend-android
echo "== JAVA =="
java -version 2>&1 | head -1
echo "== gradle build (assembleRelease) =="
./gradlew assembleRelease --no-daemon 2>&1 | tail -40
echo "== APK outputs =="
find app/build/outputs/apk -name '*.apk' -exec ls -lh {} \; 2>/dev/null
