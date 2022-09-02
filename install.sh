echo "Getting kubectl-frisbee plugin"
#!/bin/sh

if [ ! -z "${DEBUG}" ];
then set -x
fi

_detect_arch() {
    case $(uname -m) in
    amd64|x86_64) echo "amd64"
    ;;
    arm64|aarch64) echo "arm64"
    ;;
    i386) echo "386"
    ;;
    *) echo "Unsupported processor architecture";
    return 1
    ;;
     esac
}

_detect_os(){
    case $(uname) in
    Linux) echo "linux"
    ;;
    #Darwin) echo "macOS"
    #;;
    #Windows) echo "Windows"
    #;;
     esac
}

_download_url() {
        local arch="$(_detect_arch)"
        local os="$(_detect_os)"
        if [ -z "$FRISBEE_VERSION" ]
        then
            local version=`curl -s https://api.github.com/repos/carv-ics-forth/frisbee/releases/latest 2>/dev/null | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/'`
            echo https://github.com/CARV-ICS-FORTH/frisbee/releases/download/${version}/kubectl-frisbee_${version:1}_${os}_${arch}
        else
            echo https://github.com/CARV-ICS-FORTH/frisbee/releases/download/v${FRISBEE_VERSION}/kubectl-frisbee_${FRISBEE_VERSION}_${os}_${arch}
        fi
}

echo "Downloading frisbee from URL: $(_download_url)"
curl -sSLf $(_download_url) > kubectl-frisbee
chmod +x kubectl-frisbee
cp kubectl-frisbee /usr/local/bin/kubectl-frisbee


echo "kubectl-frisbee installed in:"
echo "- /usr/local/bin/kubectl-frisbee"
echo ""
echo "You'll also need 'helm' and Kubernetes 'kubectl' installed."
echo "- Install Helm: https://helm.sh/docs/intro/install/"
echo "- Install kubectl: https://kubernetes.io/docs/tasks/tools/#kubectl"