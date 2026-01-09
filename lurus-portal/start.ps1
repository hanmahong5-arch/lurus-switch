# Lurus Portal Server Startup Script
# Run with: powershell -ExecutionPolicy Bypass -File start.ps1

$env:NITRO_PORT = "3001"
$env:NODE_ENV = "production"

# Supabase configuration (update these values)
$env:SUPABASE_URL = "https://demo.supabase.co"
$env:SUPABASE_KEY = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.demo"

# Backend services
$env:BILLING_SERVICE_URL = "http://localhost:18103"
$env:SUBSCRIPTION_SERVICE_URL = "http://localhost:18104"
$env:NEW_API_URL = "http://localhost:3000"

# Site configuration
$env:SITE_URL = "http://ai.lurus.cn"

Write-Host "Starting Lurus Portal on port $env:NITRO_PORT..."
node .output/server/index.mjs
