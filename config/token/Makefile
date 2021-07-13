.PHONY: trust clean cert

trust: cert secret

cert:
	openssl genrsa -aes256 -passout pass:will_be_removed -out ./trust/key.pem 2048
	openssl rsa -in ./trust/key.pem -passin pass:will_be_removed -out ./trust/key.pem
	openssl req -new -key ./trust/key.pem -config ./trust/cert.conf -out ./trust/token-service.csr
	openssl x509 -req -days 1825 -extensions v3_user -in ./trust/token-service.csr \
    -CA ../pki/ca.crt -CAcreateserial \
    -CAkey ../pki/ca.key \
    -out  ./trust/cert.pem -extfile ./trust/cert.conf

secret:
	kubectl create secret tls registry-token-ingress -n registry-system --cert=trust/cert.pem --key=trust/key.pem

clean:
	rm -rf ./trust/*.pem ./trust/*.csr
