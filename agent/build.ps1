param(
    [string]$Output = 'source',
    [string]$Only   = ''
)

$ErrorActionPreference = 'Stop'
$root     = $PSScriptRoot                      # agent/ 自身
$checkDir = Join-Path $root 'check'
$outDir   = Join-Path $root 'dist'

Write-Host '========================================' -ForegroundColor Cyan
Write-Host '  iUpdater Agent - cross build' -ForegroundColor Cyan
Write-Host '========================================' -ForegroundColor Cyan

$useDist = $Output -eq 'dist'
if ($useDist) {
    if (Test-Path $outDir) { Remove-Item $outDir -Recurse -Force }
    New-Item $outDir -ItemType Directory -Force | Out-Null
    Copy-Item (Join-Path $root 'run.sh')             (Join-Path $outDir 'run.sh')             -Force
    Copy-Item (Join-Path $root 'config.example.json') (Join-Path $outDir 'config.example.json') -Force
}

function Build {
    param(
        [string]$GoOS,
        [string]$GoArch,
        [string]$OutName,
        [string]$SourceDir,
        [string]$Label
    )
    Write-Host "  -> $Label ..." -ForegroundColor Yellow
    Push-Location $SourceDir
    try {
        $env:CGO_ENABLED = '0'
        $env:GOOS        = $GoOS
        $env:GOARCH      = $GoArch
        go build -ldflags '-s -w' -o $OutName .
        if ($LASTEXITCODE -ne 0) {
            Write-Host "     FAILED (os=$GoOS arch=$GoArch)" -ForegroundColor Red
            exit 1
        }
        $outPath = Join-Path (Get-Location) $OutName
        $size    = if (Test-Path $outPath) { '{0:N1} MB' -f ((Get-Item $outPath).Length / 1MB) } else { '?' }
        Write-Host "     OK  $OutName  ($size)" -ForegroundColor Green
        if ($useDist) { Move-Item $OutName $outDir -Force }
    } finally {
        Pop-Location
    }
}

Write-Host ''
Write-Host '[1/2] Agent main' -ForegroundColor Cyan
$targets = @()
if ($Only -eq '' -or $Only -eq 'linux-amd64') { $targets += @{os='linux';   arch='amd64'; name='agent-linux-amd64'; label='linux/amd64' } }
if ($Only -eq '' -or $Only -eq 'linux-arm64') { $targets += @{os='linux';   arch='arm64'; name='agent-linux-arm64'; label='linux/arm64' } }
if ($Only -eq '' -or $Only -eq 'windows')     { $targets += @{os='windows'; arch='amd64'; name='agent.exe';         label='windows/amd64' } }

foreach ($t in $targets) {
    Build -GoOS $t.os -GoArch $t.arch -OutName $t.name -SourceDir $root -Label $t.label
}

Write-Host ''
Write-Host '[2/2] check validator (independent module)' -ForegroundColor Cyan
$checks = @()
if ($Only -eq '' -or $Only -eq 'linux-amd64') { $checks += @{os='linux';   arch='amd64'; name='agent-check-linux-amd64' } }
if ($Only -eq '' -or $Only -eq 'linux-arm64') { $checks += @{os='linux';   arch='arm64'; name='agent-check-linux-arm64' } }
if ($Only -eq '' -or $Only -eq 'windows')     { $checks += @{os='windows'; arch='amd64'; name='agent-check.exe' } }

foreach ($t in $checks) {
    Build -GoOS $t.os -GoArch $t.arch -OutName $t.name -SourceDir $checkDir -Label ("check " + $t.os + "/" + $t.arch)
}

Write-Host ''
Write-Host '========================================' -ForegroundColor Cyan
Write-Host '  DONE' -ForegroundColor Green
if ($useDist) {
    Write-Host "  output: $outDir"
    Get-ChildItem $outDir | ForEach-Object {
        $size = if ($_.PSIsContainer) { 'dir' } else { '{0:N1} MB' -f ($_.Length / 1MB) }
        Write-Host ('    ' + $_.Name.PadRight(30) + $size)
    }
} else {
    Write-Host '  output: agent/ + agent/check/ (source dirs)'
}
Write-Host '========================================' -ForegroundColor Cyan
