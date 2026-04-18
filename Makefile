IMAGE=queryexplorer:latest

.PHONY: deploy

deploy:
	docker build -t $(IMAGE) .
	docker save $(IMAGE) | sudo k3s ctr images import -
	kubectl rollout restart deploy/queryexplorer -n quintus
	kubectl rollout status deploy/queryexplorer -n quintus

.PHONY: demo-up demo-down

demo-up:
	docker compose -f demo-setup/docker-compose.yml up --build -d
	@echo "Demo running at http://localhost:8888"

demo-down:
	docker compose -f demo-setup/docker-compose.yml down

demo-reload:
	docker compose -f demo-setup/docker-compose.yml up -d --build app
