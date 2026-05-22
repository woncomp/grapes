param(
    [string]$InstallDir
)

Set-StrictMode -Version 2.0
$ErrorActionPreference = "Stop"
[System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor [System.Net.SecurityProtocolType]::Tls12

$Owner = "woncomp"
$Repo = "grapes"
$ApiUrl = "https://api.github.com/repos/$Owner/$Repo/releases/latest"

function New-WebClient {
    $client = New-Object System.Net.WebClient
    $client.Headers.Add("User-Agent", "grapes-install-script")
    return $client
}

function Get-Text {
    param(
        [string]$Url
    )

    $client = New-WebClient
    try {
        return $client.DownloadString($Url)
    }
    finally {
        $client.Dispose()
    }
}

function Get-File {
    param(
        [string]$Url,
        [string]$Destination
    )

    $client = New-WebClient
    try {
        $client.DownloadFile($Url, $Destination)
    }
    finally {
        $client.Dispose()
    }
}

function Get-DownloadUrls {
    param(
        [string]$Json
    )

    $urls = @()
    $matches = [regex]::Matches($Json, '"browser_download_url"\s*:\s*"([^"]+)"')
    foreach ($match in $matches) {
        $urls += $match.Groups[1].Value
    }

    return $urls
}

function Get-Os {
    if ($env:OS -eq "Windows_NT") {
        return "windows"
    }

    $uname = & uname -s 2>$null
    switch ($uname) {
        "Linux" { return "linux" }
        "Darwin" { return "darwin" }
        default { throw "Unsupported operating system: $uname" }
    }
}

function Get-Arch {
    if ($env:OS -eq "Windows_NT") {
        if ($env:PROCESSOR_ARCHITEW6432) {
            $machine = $env:PROCESSOR_ARCHITEW6432
        }
        else {
            $machine = $env:PROCESSOR_ARCHITECTURE
        }
    }
    else {
        $machine = & uname -m 2>$null
    }

    switch -Regex ($machine) {
        "^(AMD64|X86_64)$" { return "amd64" }
        "^(ARM64|AARCH64)$" { return "arm64" }
        default { throw "Unsupported architecture: $machine" }
    }
}

function Get-Sha256 {
    param(
        [string]$Path
    )

    $stream = [System.IO.File]::OpenRead($Path)
    try {
        $sha = [System.Security.Cryptography.SHA256]::Create()
        try {
            return (($sha.ComputeHash($stream) | ForEach-Object { $_.ToString("x2") }) -join "")
        }
        finally {
            $sha.Dispose()
        }
    }
    finally {
        $stream.Dispose()
    }
}

function Get-ExpectedHash {
    param(
        [string]$ChecksumsPath,
        [string]$AssetName
    )

    foreach ($line in [System.IO.File]::ReadAllLines($ChecksumsPath)) {
        if ($line -match '^(?<hash>[0-9a-fA-F]+)\s+\*?(?<name>.+)$' -and $matches["name"] -eq $AssetName) {
            return $matches["hash"].ToLowerInvariant()
        }
    }

    throw "Could not find checksum entry for $AssetName"
}

if (-not $InstallDir) {
    if ($env:GRAPES_INSTALL_DIR) {
        $InstallDir = $env:GRAPES_INSTALL_DIR
    }
    else {
        $InstallDir = Join-Path ([Environment]::GetFolderPath("LocalApplicationData")) "Microsoft\WindowsApps"
    }
}

$os = Get-Os
$arch = Get-Arch
$archiveExtension = if ($os -eq "windows") { "zip" } else { "tar.gz" }
$releaseJson = Get-Text -Url $ApiUrl
$downloadUrls = Get-DownloadUrls -Json $releaseJson

$assetUrl = $null
$checksumUrl = $null

foreach ($url in $downloadUrls) {
    $name = [System.IO.Path]::GetFileName($url)
    if ($name -like ("grapes_*_{0}_{1}.{2}" -f $os, $arch, $archiveExtension)) {
        $assetUrl = $url
    }
    elseif ($name -like "grapes_*_checksums.txt") {
        $checksumUrl = $url
    }
}

if (-not $assetUrl) {
    throw "Could not find a release asset for $os/$arch"
}

if (-not $checksumUrl) {
    throw "Could not find the release checksum file"
}

$tempRoot = Join-Path ([System.IO.Path]::GetTempPath()) ("grapes-install-" + [System.Guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Path $tempRoot | Out-Null

try {
    $assetName = [System.IO.Path]::GetFileName($assetUrl)
    $assetPath = Join-Path $tempRoot $assetName
    $checksumsPath = Join-Path $tempRoot ([System.IO.Path]::GetFileName($checksumUrl))

    Get-File -Url $assetUrl -Destination $assetPath
    Get-File -Url $checksumUrl -Destination $checksumsPath

    $expectedHash = Get-ExpectedHash -ChecksumsPath $checksumsPath -AssetName $assetName
    $actualHash = Get-Sha256 -Path $assetPath

    if ($actualHash -ne $expectedHash) {
        throw "Checksum verification failed for $assetName"
    }

    $extractDir = Join-Path $tempRoot "extract"

    if ($os -eq "windows") {
        Add-Type -AssemblyName System.IO.Compression.FileSystem
        [System.IO.Compression.ZipFile]::ExtractToDirectory($assetPath, $extractDir)
        $binaryName = "grapes.exe"
    }
    else {
        New-Item -ItemType Directory -Path $extractDir | Out-Null
        & tar -xzf $assetPath -C $extractDir
        if ($LASTEXITCODE -ne 0) {
            throw "Failed to extract $assetName"
        }
        $binaryName = "grapes"
    }

    $binary = Get-ChildItem -Path $extractDir -Recurse | Where-Object { -not $_.PSIsContainer -and $_.Name -ieq $binaryName } | Select-Object -First 1
    if (-not $binary) {
        throw "Could not find $binaryName in the extracted archive"
    }

    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    $resolvedInstallDir = (Resolve-Path -LiteralPath $InstallDir).Path
    $destination = Join-Path $resolvedInstallDir $binaryName
    Copy-Item -Path $binary.FullName -Destination $destination -Force

    Write-Host "Installed $binaryName to $destination"

    $pathSeparator = if ($os -eq "windows") { ";" } else { ":" }
    $onPath = $false
    foreach ($entry in ($env:PATH -split [regex]::Escape($pathSeparator))) {
        if ($entry -and $entry.TrimEnd("\/") -ieq $resolvedInstallDir.TrimEnd("\/")) {
            $onPath = $true
            break
        }
    }

    if (-not $onPath) {
        Write-Host "Add $resolvedInstallDir to your PATH to run grapes directly."
    }
}
finally {
    if (Test-Path $tempRoot) {
        Remove-Item -Path $tempRoot -Recurse -Force
    }
}
