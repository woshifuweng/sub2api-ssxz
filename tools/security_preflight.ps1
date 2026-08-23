[CmdletBinding()]
param(
    [switch]$SkipBuild
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

$repoRoot = Split-Path -Parent $PSScriptRoot
$backend = Join-Path $repoRoot 'backend'
$frontend = Join-Path $repoRoot 'frontend'
$auditFile = Join-Path ([System.IO.Path]::GetTempPath()) 'sub2api-pnpm-audit.json'
$requiredGoToolchain = 'go1.26.6'
$env:GOTOOLCHAIN = $requiredGoToolchain

function Invoke-Gate {
    param(
        [Parameter(Mandatory = $true)][string]$Name,
        [Parameter(Mandatory = $true)][scriptblock]$Command
    )

    Write-Host "`n==> $Name"
    & $Command
    if ($LASTEXITCODE -ne 0) {
        throw "$Name failed with exit code $LASTEXITCODE"
    }
}

try {
    $goVersion = (& go version)
    if ($LASTEXITCODE -ne 0 -or $goVersion -notmatch [regex]::Escape($requiredGoToolchain)) {
        throw "Security builds require $requiredGoToolchain; got: $goVersion"
    }
    Write-Host "Go toolchain: $goVersion"

    Invoke-Gate 'Backend vet' { Push-Location $backend; try { go vet ./... } finally { Pop-Location } }
    Invoke-Gate 'Backend tests' { Push-Location $backend; try { go test ./... } finally { Pop-Location } }
    Invoke-Gate 'Reachable Go vulnerabilities' { Push-Location $backend; try { govulncheck ./... } finally { Pop-Location } }

    Write-Host "`n==> Frontend production dependency audit"
    Push-Location $frontend
    try {
        pnpm audit --prod --audit-level=high --json | Set-Content -LiteralPath $auditFile -Encoding utf8
        # The local production-release gate is intentionally stricter than CI exceptions:
        # a deployable candidate must contain zero high/critical production findings.
        $audit = Get-Content -LiteralPath $auditFile -Raw | ConvertFrom-Json
        $high = 0
        $critical = 0
        if ($null -ne $audit.metadata -and $null -ne $audit.metadata.vulnerabilities) {
            $high = [int]$audit.metadata.vulnerabilities.high
            $critical = [int]$audit.metadata.vulnerabilities.critical
        }
        if ($high -ne 0 -or $critical -ne 0) {
            throw "Frontend production dependency audit found high=$high critical=$critical"
        }
        Write-Host 'Frontend production dependency audit: high=0 critical=0'
    }
    finally {
        Pop-Location
    }

    Invoke-Gate 'Frontend typecheck' { Push-Location $frontend; try { pnpm exec vue-tsc --noEmit } finally { Pop-Location } }
    Invoke-Gate 'Frontend tests' { Push-Location $frontend; try { pnpm exec vitest run } finally { Pop-Location } }

    if (-not $SkipBuild) {
        Invoke-Gate 'Frontend production build' { Push-Location $frontend; try { pnpm run build } finally { Pop-Location } }

        $dist = Join-Path $backend 'internal/web/dist'
        $files = @(Get-ChildItem -LiteralPath $dist -Recurse -File)
        $bytes = ($files | Measure-Object -Property Length -Sum).Sum
        $index = Join-Path $dist 'index.html'
        $indexHtml = Get-Content -LiteralPath $index -Raw
        $queryVersionCount = ([regex]::Matches($indexHtml, [regex]::Escape('?v='))).Count

        if ($files.Count -lt 200) {
            throw "Embedded frontend is incomplete: only $($files.Count) files"
        }
        if ($bytes -lt 6000000) {
            throw "Embedded frontend is unexpectedly small: $bytes bytes"
        }
        if ($queryVersionCount -ne 0) {
            throw "Frontend entry contains forbidden ?v= query strings"
        }

        Write-Host "Embedded frontend: $($files.Count) files / $bytes bytes / ?v=0"
    }

    Write-Host "`nSECURITY PREFLIGHT PASS"
}
finally {
    Remove-Item -LiteralPath $auditFile -Force -ErrorAction SilentlyContinue
}
