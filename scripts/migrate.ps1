$ErrorActionPreference = 'Stop'
if (-not $env:DATABASE_URL) { throw 'DATABASE_URL is required' }
Get-ChildItem -LiteralPath (Join-Path $PSScriptRoot '..\migrations') -Filter '*.sql' |
  Sort-Object Name |
  ForEach-Object { psql $env:DATABASE_URL -v ON_ERROR_STOP=1 -f $_.FullName }

