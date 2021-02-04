
dev.yml:
	@echo "====== Create $@ ======="
	kubectl kustomize . > dev.yml


dev: dev.yml
	@echo "====== Run target: $@ ======="
	kubectl create -n registry-system -f dev.yml

clean:
	kubectl delete -n registry-system -f dev.yml
	rm -rf dev.yml

