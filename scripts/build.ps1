param(
    [Parameter()]
    [ValidateSet('proto', 'build', 'run', 'test', 'clean')]
    [string]$Command
)

switch ($Command) {
    'proto' {
        Write-Host "Generating proto files..."
        & "$PSScriptRoot\generate_proto.ps1"
    }
    'build' {
        Write-Host "Building application..."
        go build -o bin/forum-service.exe
    }
    'run' {
        Write-Host "Running application..."
        go run main.go
    }
    'test' {
        Write-Host "Running tests..."
        go test ./...
    }
    'clean' {
        Write-Host "Cleaning build artifacts..."
        Remove-Item -Recurse -Force -ErrorAction SilentlyContinue bin/
    }
    default {
        Write-Host "Unknown command: $Command"
        Write-Host "Available commands: proto, build, run, test, clean"
    }
} 