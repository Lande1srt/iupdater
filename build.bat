@echo off
echo ========================================
echo   iUpdater Build Script
echo ========================================
echo.

echo [1/4] Checking frontend dependencies...
if not exist "frontend\node_modules" (
    echo Installing frontend dependencies...
    cd frontend
    call npm install
    if errorlevel 1 (
        echo Frontend dependency installation failed!
        pause
        exit /b 1
    )
    cd ..
) else (
    echo Frontend dependencies already exist
)
echo.

echo [2/4] Building frontend static assets...
cd frontend
call npm run build
if errorlevel 1 (
    echo Frontend build failed!
    pause
    exit /b 1
)
cd ..
echo.

echo [3/4] Copying frontend assets to backend...
if exist "backend\web\dist" (
    rmdir /s /q "backend\web\dist"
)
xcopy /e /i /y "frontend\dist" "backend\web\dist" > nul
if errorlevel 1 (
    echo Copy frontend assets failed!
    pause
    exit /b 1
)
echo Frontend assets copied to backend\web\dist
echo.

echo [4/4] Building backend executable...
cd backend
call go mod tidy
if errorlevel 1 (
    echo Go dependency tidy failed!
    pause
    exit /b 1
)

echo Building Windows version...
if not exist "dist" mkdir dist
set CGO_ENABLED=1
go build -o dist\iupdater.exe .
if errorlevel 1 (
    echo Backend build failed!
    pause
    exit /b 1
)

echo Copying config files...
if exist ".env.example" copy /y ".env.example" "dist\.env.example" > nul
if exist "data" xcopy /e /i /y "data" "dist\data" > nul 2>nul

cd ..
echo.

echo ========================================
echo   Build Complete!
echo ========================================
echo.
echo Executable: backend\dist\iupdater.exe
echo Config:     backend\dist\.env.example
echo.
echo Usage:
echo   1. Go to backend\dist directory
echo   2. Copy .env.example to .env and modify settings
echo   3. Run iupdater.exe
echo   4. Open http://localhost:8080 in browser
echo.
pause
