echo "Getting kubectl-frisbee plugin"
#!/bin/sh

if [ ! -z "${DEBUG}" ];
then set -x
fi

_detect_arch() {
    case $(uname -m) in
    amd64|x86_64) echo "x86_64"
    ;;
    arm64|aarch64) echo "arm64"
    ;;
    i386) echo "i386"
    ;;
    *) echo "Unsupported processor architecture";
    return 1
    ;;
     esac
}

_detect_os(){
    case $(uname) in
    Linux) echo "Linux"
    ;;
    Darwin) echo "macOS"
    ;;
    Windows) echo "Windows"
    ;;
     esac
}

_download_url() {
        local arch="$(_detect_arch)"
        local os="$(_detect_os)"
        if [ -z "$FRISBEE_VERSION" ]
        then
                local version=`curl -s https://api.github.com/repos/carv-ics-forth/frisbee/releases/latest 2>/dev/null | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/'`
                echo "https://github.com/carv-ics-forth/frisbee/archive/refs/tags/${version}.tar.gz"
        else
                echo "https://github.com/carv-ics-forth/frisbee/archive/refs/tags/v${FRISBEE_VERSION}.tar.gz"
        fi
}

echo "Downloading frisbee from URL: $(_download_url)"
curl -sSLf $(_download_url) > frisbee.tar.gz
tar -xzf frisbee.tar.gz kubectl-frisbee
rm frisbee.tar.gz
mv kubectl-frisbee /usr/local/bin/kubectl-frisbee
echo "kubectl-frisbee installed in /usr/local/bin/kubectl-frisbee"

