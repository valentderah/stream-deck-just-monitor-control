$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
$dist = Join-Path $root "dist"
$out = Join-Path $dist "com.valentderah.just-monitor-control.sdPlugin"

Remove-Item $out -Recurse -Force -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force -Path (Join-Path $out "bin") | Out-Null

$env:CGO_ENABLED = "0"
go build -ldflags "-s -w" -o (Join-Path $out "bin/plugin.exe") (Join-Path $root "cmd/plugin")

Copy-Item -Force (Join-Path $root "assets/manifest.json") $out
Copy-Item -Force (Join-Path $root "assets/*.json") $out -ErrorAction SilentlyContinue
Copy-Item -Recurse -Force (Join-Path $root "assets/ui") (Join-Path $out "ui")
Copy-Item -Recurse -Force (Join-Path $root "assets/imgs") (Join-Path $out "imgs")

Write-Host "Built $out"
