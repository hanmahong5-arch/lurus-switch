# Lurus Switch Test Script
# Run all tests for the project

Write-Host "Running Go tests..." -ForegroundColor Cyan
Set-Location $PSScriptRoot\..
go test -v ./...

Write-Host "`nRunning frontend type check..." -ForegroundColor Cyan
Set-Location frontend
npm run build

Write-Host "`nAll tests completed!" -ForegroundColor Green
