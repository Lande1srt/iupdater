Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  iUpdater Build Script" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

$ErrorActionPreference = "Stop"

$ProjectRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $ProjectRoot

try {
    Write-Host "[1/4] Checking frontend dependencies..." -ForegroundColor Yellow
    if (-not (Test-Path "frontend\node_modules")) {
        Write-Host "Installing frontend dependencies..." -ForegroundColor Gray
        Set-Location "frontend"
        npm install
        if ($LASTEXITCODE -ne 0) {
            throw "Frontend dependency installation failed"
        }
        Set-Location $ProjectRoot
    } else {
        Write-Host "Frontend dependencies already exist" -ForegroundColor Green
    }
    Write-Host ""

    Write-Host "[2/4] Building frontend static assets..." -ForegroundColor Yellow
    Set-Location "frontend"
    npm run build
    if ($LASTEXITCODE -ne 0) {
        throw "Frontend build failed"
    }
    Set-Location $ProjectRoot
    Write-Host ""

    Write-Host "[3/4] Copying frontend assets to backend..." -ForegroundColor Yellow
    $distPath = "backend\web\dist"
    if (Test-Path $distPath) {
        Remove-Item -Recurse -Force $distPath
    }
    Copy-Item -Recurse -Force "frontend\dist" $distPath
    Write-Host "Frontend assets copied to backend\web\dist" -ForegroundColor Green
    Write-Host ""

    Write-Host "[4/4] Building backend executable..." -ForegroundColor Yellow
    Set-Location "backend"
    
    Write-Host "Tidying Go dependencies..." -ForegroundColor Gray
    go mod tidy
    if ($LASTEXITCODE -ne 0) {
        throw "Go dependency tidy failed"
    }

    $outputDir = "dist"
    if (-not (Test-Path $outputDir)) {
        New-Item -ItemType Directory -Path $outputDir -Force | Out-Null
    }

    Write-Host "Building Windows version..." -ForegroundColor Gray
    $env:CGO_ENABLED = "1"
    go build -o "$outputDir\iupdater.exe" .
    if ($LASTEXITCODE -ne 0) {
        throw "Backend build failed"
    }

    Write-Host "Copying config files and data directory..." -ForegroundColor Gray
    if (Test-Path ".env.example") {
        Copy-Item ".env.example" "$outputDir\.env.example" -Force
    }
    if (Test-Path "data") {
        Copy-Item -Recurse -Force "data" "$outputDir\data" -ErrorAction SilentlyContinue
    }

    Set-Location $ProjectRoot
    Write-Host ""

    Write-Host "========================================" -ForegroundColor Green
    Write-Host "  Build Complete!" -ForegroundColor Green
    Write-Host "========================================" -ForegroundColor Green
    Write-Host ""
    Write-Host "Executable: backend\dist\iupdater.exe" -ForegroundColor White
    Write-Host "Config:     backend\dist\.env.example" -ForegroundColor White
    Write-Host ""
    Write-Host "Usage:" -ForegroundColor Yellow
    Write-Host "  1. Go to backend\dist directory"
    Write-Host "  2. Copy .env.example to .env and modify settings"
    Write-Host "  3. Run iupdater.exe"
    Write-Host "  4. Open http://localhost:8080 in browser"
    Write-Host ""

} catch {
    Write-Host ""
    Write-Host "Build failed: $($_.Exception.Message)" -ForegroundColor Red
    Set-Location $ProjectRoot
    exit 1
}
