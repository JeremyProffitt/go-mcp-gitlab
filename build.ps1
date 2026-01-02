$APP_NAME = "go-mcp-gitlab"
$VERSION = "1.0.0"

Write-Host "Building $APP_NAME version $VERSION..."
go build -ldflags "-X main.Version=$VERSION" -o "$APP_NAME.exe" .

if ($LASTEXITCODE -eq 0) {
    Write-Host "Build successful: $APP_NAME.exe"
} else {
    Write-Host "Build failed"
    exit 1
}
