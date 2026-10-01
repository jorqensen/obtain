# Installation

## Install script (Linux and macOS)

```sh
curl -fsSL https://raw.githubusercontent.com/jorqensen/obtain/main/install.sh | sh
```

The script detects your OS and architecture, downloads the matching binary from the latest [GitHub release](https://github.com/jorqensen/obtain/releases), verifies its checksum and installs it to `~/.local/bin/obtain`. If that directory isn't on your `PATH`, it tells you how to add it.

Two environment variables adjust it:

| Variable | Description | Default |
| --- | --- | --- |
| `OBTAIN_VERSION` | Install a specific version, e.g. `v0.1.0` | latest |
| `OBTAIN_INSTALL_DIR` | Directory to install the binary into | `~/.local/bin` |

```sh
curl -fsSL https://raw.githubusercontent.com/jorqensen/obtain/main/install.sh | OBTAIN_VERSION=v0.1.0 sh
```

## Windows

The install script doesn't support Windows, so download the `obtain_<version>_windows_<arch>.zip` from the [releases page](https://github.com/jorqensen/obtain/releases) and put `obtain.exe` in a directory on your `PATH`. Windows has no standard per-user bin directory, so create one. `%LOCALAPPDATA%\Programs\obtain` is a good choice, since it needs no admin rights.

In PowerShell (use `amd64` or `arm64` to match your machine, and set the version you want):

```powershell
$version = "0.1.0"
$arch = "amd64"
$dir = "$env:LOCALAPPDATA\Programs\obtain"

Invoke-WebRequest "https://github.com/jorqensen/obtain/releases/download/v$version/obtain_${version}_windows_$arch.zip" -OutFile "$env:TEMP\obtain.zip"
Expand-Archive "$env:TEMP\obtain.zip" -DestinationPath $dir -Force
Remove-Item "$env:TEMP\obtain.zip"

# Add the directory to your user PATH (once)
$path = [Environment]::GetEnvironmentVariable("Path", "User")
if ($path -notlike "*$dir*") {
  [Environment]::SetEnvironmentVariable("Path", "$path;$dir", "User")
}
```

Open a new terminal afterwards, since existing ones don't pick up the changed `PATH`. Then check with `obtain --version`.

The default config location is `$XDG_CONFIG_HOME/obtain/config.yaml`, falling back to `~/.config/obtain/config.yaml`, which on Windows is `%USERPROFILE%\.config\obtain\config.yaml`.

## With Go

```sh
go install github.com/jorqensen/obtain@latest
```

## Manual download

Prebuilt binaries for Linux, macOS and Windows (amd64, arm64) are attached to each [GitHub release](https://github.com/jorqensen/obtain/releases). Extract the archive and put `obtain` somewhere on your `PATH`.
