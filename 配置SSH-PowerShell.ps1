# PowerShell script to configure SSH key
# Run this in PowerShell (as Administrator if needed)

$ServerIP = "115.190.239.146"
$User = "root"
$PublicKey = Get-Content "$env:USERPROFILE\.ssh\id_rsa.pub" -Raw

Write-Host "=== Configuring SSH Key Authentication ===" -ForegroundColor Green
Write-Host ""
Write-Host "Server: $ServerIP" -ForegroundColor Cyan
Write-Host "User: $User" -ForegroundColor Cyan
Write-Host ""
Write-Host "You will be prompted for password: GGsuperman1211" -ForegroundColor Yellow
Write-Host ""

# Create the command to execute on server
$command = "mkdir -p ~/.ssh && chmod 700 ~/.ssh && echo '$PublicKey' >> ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys && echo 'SSH key configured successfully!'"

# Execute via SSH
ssh -o StrictHostKeyChecking=no ${User}@${ServerIP} "$command"

if ($LASTEXITCODE -eq 0) {
    Write-Host ""
    Write-Host "=== SSH Key Configuration Complete ===" -ForegroundColor Green
    Write-Host ""
    Write-Host "Testing connection (should not require password)..." -ForegroundColor Cyan

    ssh ${User}@${ServerIP} "echo 'Connection test successful!'"

    if ($LASTEXITCODE -eq 0) {
        Write-Host ""
        Write-Host "SUCCESS! SSH key authentication is working!" -ForegroundColor Green
        Write-Host ""
        Write-Host "Now you can run the deployment script:" -ForegroundColor Yellow
        Write-Host "  cd D:\tools\lurus-switch" -ForegroundColor White
        Write-Host "  bash deploy-now.sh" -ForegroundColor White
        Write-Host ""
    }
} else {
    Write-Host ""
    Write-Host "Configuration failed. Please check the error message above." -ForegroundColor Red
}
