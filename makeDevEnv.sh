#!/bin/bash

set -eu;

## GLOBAL VARIABLES
__dir="$(cd "$(dirname "${0}")" && pwd)"
__user="${SUDO_USER:-$USER}"

__escEnv="homelab-dev"
__vmProvider="orb"
__vmProviderTimeoutSeconds=180
__vmName="k3s-master-dev"

waitForConnectivity() {
  local hostname="${1}" && shift;
  local port="${1}" && shift;

  local counter=0;
  until nc -zv "${hostname}" "${port}" >/dev/null 2>&1; do
    ((counter++))
    if [ "${counter}" -ge "${__vmProviderTimeoutSeconds}" ]; then
      printf '\n    • Too much waiting something should not have worked properly\n';
      exit 1;
    fi
    sleep 1;
    printf '.'
  done
}

mkCloudInitMaster() {
  esc run "${__escEnv}" --interactive -- envsubst < "${__dir}/cloud-configs/k3s-master.yaml.tmpl" > "${__dir}/cloud-configs/k3s-master.yaml";
}

catVMFile() {
  local vmFilePath="${1}" && shift;

  case ${__vmProvider} in
    orb)
    orb -m "${__vmName}" -u root cat "${vmFilePath}"
    ;;

    *)
    printf 'Unknown Provider : %s' "${__vmProvider}";
    ;;
  esac
}

saveK3SToken() {
  local k3sToken="$(catVMFile "/var/lib/rancher/k3s/server/agent-token")";
  local escK3SToken="$(esc env get "${__escEnv}" "k3s.token" --show-secrets --value json | yq .)";

  # Only save when value in ESC do not match to avoid increasing version everytime.
  if [ "${k3sToken}" != "${escK3SToken}" ]; then
    esc env set "${__escEnv}" "k3s.token" "${k3sToken}" --secret;
  else
    echo "Actual token matches the ESC one, doing nothing.";
  fi
}

saveKubeconfig() {
  local hostname="$(esc env get "${__escEnv}" "k3s.nodes.master" --show-secrets --value json | yq .)";
  local tailnet="$(esc env get "${__escEnv}" "tailscale.tailnet" --show-secrets --value json | yq .)";
  local port="$(esc env get "${__escEnv}" "k3s.config.httpsListenPort" --show-secrets --value json | yq .)";
  local k3sMasterURI="$(printf 'https://%s.%s.ts.net:%s' "${hostname}" "${tailnet}" "${port}")";

  local dir="$(mktemp -d)";

  catVMFile '/etc/rancher/k3s/k3s.yaml' | \
      yq --no-colors --output-format=yaml --indent=2 --prettyPrint ".clusters[0].cluster.server = \"${k3sMasterURI}\"" \
      > "${dir}/kubeconfigVM";

  esc env get "${__escEnv}" "k3s.kubeconfig" --show-secrets --value json | yq '.' | \
      yq --no-colors --output-format=yaml --indent=2 --prettyPrint '.' \
      > "${dir}/kubeconfigESC";

  if [ "$(<"${dir}/kubeconfigVM")" != "$(<"${dir}/kubeconfigESC")" ]; then
    local kubeconfig="$(yq --input-format yaml --output-format json --indent=0 '.' "${dir}/kubeconfigVM")";
    esc env set "${__escEnv}" --secret "k3s.kubeconfig" \'"${kubeconfig}"\';
  else
    echo "Actual kubeconfig matches the ESC one, doing nothing.";
  fi

  rm -f "${dir}/kubeconfigVM" "${dir}/kubeconfigESC";
  rmdir "${dir}";
}

mkMaster() {
  local cloudInitFilePath="${__dir}/cloud-configs/k3s-master.yaml";

  if [ ! -f "${cloudInitFilePath}" ]; then
    printf "ERROR file '%s' expected but not found" "${cloudInitFilePath}"
    exit 1
  fi
  case ${__vmProvider} in
    orb)
    if orb list --quiet | grep "${__vmName}" > /dev/null; then
      printf '    • A VM in Orbstack with name "%s" already exists: ... trying to start the VM.\n' "${__vmName}";
      orb start "${__vmName}";
    else
      printf '    • Creating a new VM in Orbstack with name "%s" using Cloud-init file "%s"\n' "${__vmName}" "${cloudInitFilePath}";
      orb create ubuntu "${__vmName}" -c "${cloudInitFilePath}";
    fi
    ;;

    *)
    printf 'Unknown Provider : %s' "${__vmProvider}";
    ;;
  esac
}

main() {
  printf 'Creating a local dev environment\n';
  printf '1)  Generating Cloud Config files for env "%s"\n' "${__escEnv}";
  mkCloudInitMaster;
  printf '    Done\n\n';

  printf '2)  Creating the K8S master VM with provider "%s":\n' "${__vmProvider}";
  mkMaster;
  printf '    • Now, waiting for Tailscale and K3S to be ready ...';
  local hostname="$(esc env get "${__escEnv}" "k3s.nodes.master" --show-secrets --value json | yq .)";
  local tailnet="$(esc env get "${__escEnv}" "tailscale.tailnet" --show-secrets --value json | yq .)";
  local port="$(esc env get "${__escEnv}" "k3s.config.httpsListenPort" --show-secrets --value json | yq .)";
  waitForConnectivity "${hostname}.${tailnet}.ts.net" "${port}";
  printf ' Found it!\n';
  printf '    Done\n\n';

  printf '3)  Saving K3S join token in the pulumi environment "%s" ... ' "${__escEnv}";
  saveK3SToken;
  printf '    Done\n\n';

  printf '4)  Saving the Kubeconfig in the pulumi environment "%s" ... ' "${__escEnv}";
  saveKubeconfig;
  printf '    Done\n\n';
}

if [ "$0" = "$BASH_SOURCE" ]; then
  main "$@"
fi
