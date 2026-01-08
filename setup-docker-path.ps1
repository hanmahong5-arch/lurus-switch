# Update Docker PATH to D:\docker
$currentPath = [Environment]::GetEnvironmentVariable('Path', 'Machine')
$newPath = $currentPath -replace 'C:\\Program Files\\docker', 'D:\docker'
[Environment]::SetEnvironmentVariable('Path', $newPath, 'Machine')
Write-Host "PATH updated. Docker now at D:\docker"

# Register Docker service
& 'D:\docker\dockerd.exe' --register-service
Write-Host "Docker service registered"

# Start Docker service
Start-Service docker
Write-Host "Docker service started"

# Verify
& 'D:\docker\docker.exe' version
