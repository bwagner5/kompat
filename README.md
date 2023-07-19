
# kompat

Kompat is a simple CLI tool to interact with `compatibility.yaml` files which host your Kubernetes compatibility matrix.

## Usage:

```
Put Usage here
Usage:
  kompat [command]
...
```

## Installation:

```
brew install bwagner5/wagner/kompat
```

Packages, binaries, and archives are published for all major platforms (Mac amd64/arm64 & Linux amd64/arm64):

Debian / Ubuntu:

```
[[ `uname -m` == "aarch64" ]] && ARCH="arm64" || ARCH="amd64"
OS=`uname | tr '[:upper:]' '[:lower:]'`
wget https://github.com/bwagner5/kompat/releases/download/v0.0.1/kompat_0.0.1_${OS}_${ARCH}.deb
dpkg --install kompat_0.0.1_linux_amd64.deb
kompat --help
```

RedHat:

```
[[ `uname -m` == "aarch64" ]] && ARCH="arm64" || ARCH="amd64"
OS=`uname | tr '[:upper:]' '[:lower:]'`
rpm -i https://github.com/bwagner5/kompat/releases/download/v0.0.1/kompat_0.0.1_${OS}_${ARCH}.rpm
```

Download Binary Directly:

```
[[ `uname -m` == "aarch64" ]] && ARCH="arm64" || ARCH="amd64"
OS=`uname | tr '[:upper:]' '[:lower:]'`
wget -qO- https://github.com/bwagner5/kompat/releases/download/v0.0.1/kompat_0.0.1_${OS}_${ARCH}.tar.gz | tar xvz
chmod +x kompat
```

## Examples: 

EXAMPLES HERE