#!/bin/sh

set -o pipefail && go run *.go || exit 1

# set -o pipefail && openssl pkcs12 -in Certificates.p12 -clcerts -nokeys -out passcertificate.pem -passin pass:yes || exit 1
# set -o pipefail && openssl pkcs12 -in Certificates.p12 -nocerts -out passkey.pem -passin pass:yes -passout pass:y || exit 1
# set -o pipefail && openssl smime -binary -sign -certfile "Apple Worldwide Developer Relations Certification Authority.pem" \
#     -signer passcertificate.pem -inkey passkey.pem -in manifest.json -out signature -outform DER -passin pass:y || exit 1

# zip -r Generic.pkpass manifest.json Generic.pass/pass.json signature Generic.pass/logo.png Generic.pass/logo@2x.png \
#     Generic.pass/icon.png Generic.pass/icon@2x.png Generic.pass/thumbnail.png thumbnail@2x.png

# ./signpass -p Generic.pass

# open Generic.pkpass