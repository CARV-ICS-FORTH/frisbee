#!/bin/bash


set -o errexit
set -o nounset
set -o pipefail

# Since all components interacting with ApiServer in Kube require bi-directional TLS authentication with ApiServer,
# we need to manually issue a self-signed CA certificate here first.

# CREATE THE PRIVATE KEY FOR OUR CUSTOM CA
openssl genrsa -out certs/ca.key 2048

# GENERATE A CA CERT WITH THE PRIVATE KEY
openssl req -x509 -new -nodes -key certs/ca.key -subj "/CN=139.91.92.82" -out certs/ca.crt

# CREATE THE PRIVATE KEY FOR OUR FRISBEE SERVER
openssl genrsa -out certs/tls.key 2048

# Crete the specs for the certificate
cat << EOF > certs/csr.conf
  [ req ]
  default_bits       = 2048
  prompt             = no
  default_md         = sha512
  distinguished_name = req_distinguished_name
  req_extensions     = req_ext


  # distinguished_name
  [ req_distinguished_name ]
  C = "GR"                        # Country
  ST = "Crete"                    #  State
  L = "Heraklion"                 # City
  O = "FORTH"                     # Organization
  OU = "CARV"                     # Organization Unit
  CN = 139.91.92.82               # DNS CommonName (frisbee.dev)

  [ req_ext ]
  subjectAltName = @alt_names

  [ alt_names ]
  IP.1 = 139.91.92.82
  DNS.1 = frisbee
  DNS.2 = frisbee.default
  DNS.3 = frisbee.default.svc
  DNS.4 = frisbee.default.svc.cluster.local

  [ v3_req ]
  authorityKeyIdentifier = keyid:always,issuer:always
  basicConstraints=CA:FALSE
  keyUsage=keyEncipherment,dataEncipherment
  extendedKeyUsage=serverAuth,clientAuth
  subjectAltName=@alt_names
EOF


# CREATE A CSR FROM THE CONFIGURATION FILE AND OUR PRIVATE KEY
openssl req -new -key certs/tls.key -out certs/server.csr -config certs/csr.conf

# CREATE THE CERT SIGNING THE CSR WITH THE CA CREATED BEFORE
openssl x509 -req -in certs/server.csr -CA certs/ca.crt -CAkey certs/ca.key -CAcreateserial -out certs/tls.crt -extfile certs/csr.conf


# Our validation webhook configuration must contain an encoded certificate authority.
# INJECT CA IN THE WEBHOOK CONFIGURATION


# In the next step, we need to create a secret to place the certificates.
#kubectl create secret generic grumpy -n default \
#  --from-file=key.pem=certs/grumpy-key.pem \
#  --from-file=cert.pem=certs/grumpy-crt.pem

#export CA_BUNDLE=$(cat certs/ca.crt | base64 | tr -d '\n')
# ahelm upgrade --install  my-frisbee ./charts/platform/ --set operator.enabled=false -f ./charts/platform/values-baremetal.yaml  --debug --set operator.webhook.k8s.caBundle="$CA_BUNDLE" | less
#cat ${WEBHOOK_DIR}/manifests.yaml | envsubst > /tmp/manifests.yaml
#mv /tmp/manifests.yaml ${WEBHOOK_DIR}/manifests.yaml