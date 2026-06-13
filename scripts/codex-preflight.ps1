param(
    [string]$Path = "."
)

$ErrorActionPreference = "SilentlyContinue"

$resolved = Resolve-Path -LiteralPath $Path
if (-not $resolved) {
    Write-Host "Path not found: $Path"
    exit 1
}

Set-Location -LiteralPath $resolved

Write-Host ""
Write-Host "== Codex Preflight =="
Write-Host "Path: $resolved"
Write-Host "Mode: read-only"
Write-Host ""

function Has-Command($Name) {
    return [bool](Get-Command $Name -ErrorAction SilentlyContinue)
}

if (Has-Command "git") {
    $branch = git rev-parse --abbrev-ref HEAD 2>$null
    if ($LASTEXITCODE -eq 0) {
        $head = git log -1 --oneline 2>$null
        Write-Host "Git branch: $branch"
        Write-Host "HEAD: $head"
        Write-Host ""
        Write-Host "Changed files:"
        $status = git status --short
        if ($status) {
            $status
        } else {
            Write-Host "  none"
        }
        Write-Host ""
    } else {
        Write-Host "Git: not a repository"
        Write-Host ""
    }
} else {
    Write-Host "Git: unavailable"
    Write-Host ""
}

$stack = New-Object System.Collections.Generic.List[string]
if (Test-Path "backend/go.mod") { $stack.Add("Go backend") }
if (Test-Path "go.mod") { $stack.Add("Go") }
if (Test-Path "frontend/package.json") { $stack.Add("Vue/TypeScript frontend") }
if (Test-Path "package.json") { $stack.Add("Node.js / JavaScript / TypeScript") }
if (Test-Path "pyproject.toml") { $stack.Add("Python") }
if (Test-Path "requirements.txt") { $stack.Add("Python") }
if (Test-Path "Cargo.toml") { $stack.Add("Rust") }

Write-Host "Detected stack:"
if ($stack.Count -gt 0) {
    $stack | Sort-Object -Unique | ForEach-Object { Write-Host "  - $_" }
} else {
    Write-Host "  unknown"
}
Write-Host ""

Write-Host "Likely verification commands:"
if (Test-Path "backend/go.mod") {
    Write-Host "  cd backend; go test ./internal/service ./internal/handler ./internal/server/routes ./cmd/server"
}
if (Test-Path "go.mod") {
    Write-Host "  go test ./..."
}
if (Test-Path "frontend/package.json") {
    $pkg = Get-Content "frontend/package.json" -Raw | ConvertFrom-Json
    if ($pkg.scripts) {
        $scripts = $pkg.scripts.PSObject.Properties.Name
        foreach ($name in @("test", "test:run", "typecheck", "lint", "build")) {
            if ($scripts -contains $name) {
                Write-Host "  cd frontend; pnpm run $name"
            }
        }
    }
}
if (Test-Path "package.json") {
    $pkg = Get-Content "package.json" -Raw | ConvertFrom-Json
    if ($pkg.scripts) {
        $scripts = $pkg.scripts.PSObject.Properties.Name
        foreach ($name in @("test", "typecheck", "lint", "build")) {
            if ($scripts -contains $name) {
                Write-Host "  npm run $name"
            }
        }
    }
}
if (Test-Path "pyproject.toml") { Write-Host "  python -m pytest" }
if (Test-Path "Cargo.toml") { Write-Host "  cargo test" }
Write-Host "  git diff --check"
Write-Host ""

Write-Host "Sensitive files present within depth 4:"
$sensitive = Get-ChildItem -Force -File -Recurse -Depth 4 |
    Where-Object {
        $path = $_.FullName
        if ($path -match '\\(\.git|node_modules|\.pnpm-store|dist|build|coverage)\\') {
            return $false
        }
        $_.Name -match '^\.env($|\.|_)' -or
        $_.Name -match '\.(pem|key|p12|pfx)$' -or
        $_.Name -match '(?i)(secret|credential|private)'
    } |
    Select-Object -First 30

if ($sensitive) {
    $sensitive | ForEach-Object { Write-Host "  - $($_.FullName)" }
    Write-Host "  Treat these as read-only unless the user explicitly asks."
} else {
    Write-Host "  none found"
}
Write-Host ""

Write-Host "Workspace reminders:"
Write-Host "  - Real model catalog entries must come from channel/account/group supported_models."
Write-Host "  - Env allowlists are filters, not model catalog sources."
Write-Host "  - Fake models must be fake/test-only and never production defaults."
Write-Host "  - Do not touch billing, ledger, payment, provider routing, Nginx, or production unless explicitly requested."
Write-Host ""
Write-Host "Done. This script made no changes."
