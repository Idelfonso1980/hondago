# check_encoding.ps1
# Uso: powershell -ExecutionPolicy Bypass -File .\scripts\check_encoding.ps1 -Path .
param(
  [string]$Path = "."
)

Get-ChildItem -Path $Path -Recurse -File |
  Where-Object { $_.Extension -in '.go','.js','.md','.sql','.yml','.yaml','.ini','.env','.ps1','.sh','.txt','.html','.css' } |
  ForEach-Object {
    try {
      $bytes = [System.IO.File]::ReadAllBytes($_.FullName)
      $text = [System.Text.Encoding]::UTF8.GetString($bytes)
      if ($text -match "�") {
        Write-Output "Possível problema de encoding: $($_.FullName)"
      }
    } catch {
      Write-Output "Falha ao ler: $($_.FullName)"
    }
  }
