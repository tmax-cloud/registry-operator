
dev.yml:
	@echo "====== Create $@ ======="
	kubectl kustomize . > dev.yml


dev: dev.yml
	@echo "====== Run target: $@ ======="
	kubectl create -f dev.yml

deploy:
	sshpass -p 'tmax@23' scp dev.yml root@220.90.208.100:/root/dev.yml
	sshpass -p 'tmax@23' ssh root@220.90.208.100 "kubectl delete -n efk -f /root/dev.yml && kubectl create -n efk -f /root/dev.yml"

clean:
	kubectl delete -f dev.yml
	rm -rf dev.yml

