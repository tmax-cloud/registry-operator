
dev.yml:
	@echo "====== Create $@ ======="
	kubectl kustomize . > dev.yml


dev: dev.yml
	@echo "====== Run target: $@ ======="
	kubectl create -f dev.yml

clean:
	kubectl delete -f dev.yml
	rm -rf dev.yml

