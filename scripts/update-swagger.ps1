$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $PSScriptRoot
Set-Location $repoRoot

Write-Host "Regenerando Swagger docs..."
go run github.com/swaggo/swag/cmd/swag@v1.16.6 init -g main.go -o docs
