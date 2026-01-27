param(
  [string]$Source = ".\schepass.exe",
  [string]$InstallDir = "$env:LOCALAPPDATA\schepass\bin",
  [string]$DefaultMessages = ".\resources\default_messages.json",
  [string]$ConfigDir = "$env:USERPROFILE\.schepass",
  [switch]$AddToPath
)

if (!(Test-Path $Source)) {
  Write-Error "Build the binary first (expected $Source)"
  exit 1
}

New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
Copy-Item -Force $Source (Join-Path $InstallDir "schepass.exe")

if ((Test-Path $DefaultMessages) -and !(Test-Path (Join-Path $ConfigDir "messages.json"))) {
  New-Item -ItemType Directory -Path $ConfigDir -Force | Out-Null
  Copy-Item -Force $DefaultMessages (Join-Path $ConfigDir "messages.json")
}

if ($AddToPath) {
  $current = [Environment]::GetEnvironmentVariable("Path", "User")
  if ($current -notlike "*$InstallDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$current;$InstallDir", "User")
    Write-Host "Added to PATH. Restart your terminal."
  }
}

Write-Host "Installed to $InstallDir\schepass.exe"
