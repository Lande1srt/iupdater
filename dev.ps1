# ========================================
#   iUpdater Dev Environment Launcher
#   一键启动开发环境（Go 后端 + Vite 前端）
# ========================================

$ErrorActionPreference = "Continue"
$ProjectRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $ProjectRoot

$backendDir = Join-Path $ProjectRoot "backend"
$frontendDir = Join-Path $ProjectRoot "frontend"
$backendProc = $null
$frontendProc = $null

function Cleanup {
    Write-Host ""
    Write-Host "Shutting down..." -ForegroundColor Red
    if ($backendProc -and -not $backendProc.HasExited) {
        Stop-Process -Id $backendProc.Id -Force -ErrorAction SilentlyContinue
    }
    if ($frontendProc -and -not $frontendProc.HasExited) {
        Stop-Process -Id $frontendProc.Id -Force -ErrorAction SilentlyContinue
    }
    Write-Host "All processes stopped." -ForegroundColor Green
}

# ========== 1. 检查依赖 ==========
Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  iUpdater Dev Environment Launcher" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "[1/5] Checking dependencies..." -ForegroundColor Yellow

$goVer = & go version 2>&1
if ($LASTEXITCODE -ne 0) { Write-Host "ERROR: Go not found" -ForegroundColor Red; exit 1 }
Write-Host "  Go: $goVer" -ForegroundColor Green

$nodeVer = & node -v 2>&1
if ($LASTEXITCODE -ne 0) { Write-Host "ERROR: Node.js not found" -ForegroundColor Red; exit 1 }
Write-Host "  Node.js: $nodeVer" -ForegroundColor Green

$npmVer = & npm -v 2>&1
if ($LASTEXITCODE -ne 0) { Write-Host "ERROR: npm not found" -ForegroundColor Red; exit 1 }
Write-Host "  npm: $npmVer" -ForegroundColor Green

# ========== 2. 准备后端配置 ==========
Write-Host ""
Write-Host "[2/5] Preparing backend configuration..." -ForegroundColor Yellow

# 端口冲突检测
Write-Host "  Checking ports 8080 and 5173..." -ForegroundColor Gray
foreach ($portNum in 8080, 5173) {
    $conn = Get-NetTCPConnection -LocalPort $portNum -State Listen -ErrorAction SilentlyContinue
    if ($conn) {
        Write-Host "  Port $portNum is in use by PID $($conn.OwningProcess), freeing..." -ForegroundColor Yellow
        Stop-Process -Id $conn.OwningProcess -Force -ErrorAction SilentlyContinue
        Start-Sleep -Milliseconds 500
    }
}

$envFile = Join-Path $backendDir ".env"
$envExample = Join-Path $backendDir ".env.example"
$dataDir = Join-Path $backendDir "data"

if (-not (Test-Path $envFile)) {
    if (Test-Path $envExample) {
        Copy-Item $envExample $envFile
        Write-Host "  Created backend/.env from .env.example" -ForegroundColor Green
    } else {
        "PORT=8080" | Out-File -FilePath $envFile -Encoding utf8
        "ADMIN_USER=admin" | Out-File -FilePath $envFile -Encoding utf8 -Append
        "ADMIN_PASS=admin123" | Out-File -FilePath $envFile -Encoding utf8 -Append
        "JWT_SECRET=iupdater-secret-key-change-in-production" | Out-File -FilePath $envFile -Encoding utf8 -Append
        "DB_PATH=data/iupdater.db" | Out-File -FilePath $envFile -Encoding utf8 -Append
        Write-Host "  Created backend/.env with defaults" -ForegroundColor Green
    }
} else {
    Write-Host "  backend/.env already exists" -ForegroundColor Green
}

if (-not (Test-Path $dataDir)) {
    New-Item -ItemType Directory -Path $dataDir -Force | Out-Null
    Write-Host "  Created backend/data directory" -ForegroundColor Green
}

# ========== 3. 检查前端依赖 ==========
Write-Host ""
Write-Host "[3/5] Checking frontend dependencies..." -ForegroundColor Yellow

if (-not (Test-Path (Join-Path $frontendDir "node_modules"))) {
    Write-Host "  Installing npm dependencies..." -ForegroundColor Gray
    Push-Location $frontendDir
    npm install
    if ($LASTEXITCODE -ne 0) {
        Write-Host "ERROR: npm install failed" -ForegroundColor Red
        Pop-Location
        exit 1
    }
    Pop-Location
    Write-Host "  npm dependencies installed" -ForegroundColor Green
} else {
    Write-Host "  node_modules already exists" -ForegroundColor Green
}

# ========== 4. 启动后端 ==========
Write-Host ""
Write-Host "[4/5] Starting Go backend..." -ForegroundColor Yellow

$backendProc = Start-Process -FilePath "go" -ArgumentList "run","." -WorkingDirectory $backendDir -NoNewWindow -PassThru

Write-Host "  Backend PID: $($backendProc.Id)" -ForegroundColor Green
Write-Host "  Waiting for backend..." -ForegroundColor Gray

$backendReady = $false
for ($i = 0; $i -lt 30; $i++) {
    Start-Sleep -Seconds 1
    if ($backendProc.HasExited) {
        Write-Host "  ERROR: Backend exited unexpectedly!" -ForegroundColor Red
        Cleanup
        exit 1
    }
    try {
        $null = Invoke-WebRequest -Uri "http://localhost:8080/" -UseBasicParsing -ErrorAction Stop
        $backendReady = $true
        break
    } catch [System.Net.WebException] {
        $resp = $_.Exception.Response
        if ($resp) {
            $code = [int]$resp.StatusCode
            if ($code -ge 200 -and $code -le 500) {
                $backendReady = $true
                break
            }
        }
    } catch {}
}

if ($backendReady) {
    Write-Host "  Backend ready on http://localhost:8080" -ForegroundColor Green
} else {
    Write-Host "  Warning: Backend health check timed out" -ForegroundColor Yellow
}

# ========== 5. 启动前端 ==========
Write-Host ""
Write-Host "[5/5] Starting Vite dev server..." -ForegroundColor Yellow

$frontendProc = Start-Process -FilePath "cmd" -ArgumentList "/c","npm","run","dev" -WorkingDirectory $frontendDir -NoNewWindow -PassThru

Write-Host "  Frontend PID: $($frontendProc.Id)" -ForegroundColor Green

$frontendReady = $false
for ($i = 0; $i -lt 15; $i++) {
    Start-Sleep -Seconds 1
    if ($frontendProc.HasExited) {
        Write-Host "  ERROR: Frontend exited unexpectedly!" -ForegroundColor Red
        Cleanup
        exit 1
    }
    try {
        $null = Invoke-WebRequest -Uri "http://localhost:5173" -UseBasicParsing -ErrorAction Stop
        $frontendReady = $true
        break
    } catch {}
}

# ========== 状态显示 ==========
Write-Host ""
Write-Host "========================================" -ForegroundColor Green
Write-Host "  Dev Environment is Running!" -ForegroundColor Green
Write-Host "========================================" -ForegroundColor Green
Write-Host ""
Write-Host "  Frontend:  http://localhost:5173" -ForegroundColor White
Write-Host "  Backend:   http://localhost:8080" -ForegroundColor White
Write-Host "  API Proxy: /api -> http://localhost:8080" -ForegroundColor Gray
Write-Host ""
Write-Host "  Default Login: admin / admin123" -ForegroundColor Yellow
Write-Host ""
Write-Host "  Press Ctrl+C to stop all services" -ForegroundColor Red
Write-Host ""

if ($frontendReady) {
    Start-Sleep -Milliseconds 800
    Start-Process "http://localhost:5173"
    Write-Host "  Browser opened automatically." -ForegroundColor Green
}

# ========== 主循环 ==========
try {
    while ($true) {
        Start-Sleep -Seconds 2
        if ($backendProc.HasExited -or $frontendProc.HasExited) {
            Write-Host ""
            Write-Host "A process has exited." -ForegroundColor Red
            break
        }
    }
} finally {
    Cleanup
}
