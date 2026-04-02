Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path

if (Get-Command py -ErrorAction SilentlyContinue) {
  & py -3 (Join-Path $scriptDir 'diagnose_github_actions.py') @args
  exit $LASTEXITCODE
}

if (Get-Command python -ErrorAction SilentlyContinue) {
  & python (Join-Path $scriptDir 'diagnose_github_actions.py') @args
  exit $LASTEXITCODE
}

throw 'python is required to diagnose GitHub Actions'
