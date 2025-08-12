#!/bin/bash

set -e

TARGET_DIR="bin"
ZIP_NAME="ccagent-beta.zip"
TEMP_DIR=$(mktemp -d)

function create_build {
    GOOS=$1
    GOARCH=$2
    EXT=$3
    if [ -z "$EXT" ]; then
        EXT=$GOARCH
    fi

    BINARY=ccagent-beta-$GOOS-$EXT
    echo "Building $BINARY..."
    GOOS=$GOOS GOARCH=$GOARCH go build -o $TEMP_DIR/$BINARY cmd/*.go
    cd $TEMP_DIR && shasum -a 256 $BINARY > $BINARY.sha256 && cd - > /dev/null
}

echo "Creating production binaries for ccagent..."

# Ensure target directory exists
mkdir -p $TARGET_DIR

# Build for all platforms
create_build windows amd64 x86_64.exe
create_build darwin amd64 x86_64
create_build linux amd64 x86_64
create_build linux arm64

# Create zip archive
echo "Creating zip archive..."
cd $TEMP_DIR
zip -r $ZIP_NAME * > /dev/null
cd - > /dev/null

# Move zip to target directory and cleanup
mv $TEMP_DIR/$ZIP_NAME $TARGET_DIR/
rm -rf $TEMP_DIR

echo "Production binaries created: $TARGET_DIR/$ZIP_NAME"