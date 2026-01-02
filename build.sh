#!/bin/bash
APP_NAME="go-mcp-gitlab"
VERSION="1.0.0"

echo "Building $APP_NAME version $VERSION..."
go build -ldflags "-X main.Version=$VERSION" -o "$APP_NAME" .

if [ $? -eq 0 ]; then
    echo "Build successful: $APP_NAME"
else
    echo "Build failed"
    exit 1
fi
