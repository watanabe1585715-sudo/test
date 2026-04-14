# Docker が使える前提。`make up` で DB+API、`make batch` で掲載反映。
# Go のみホスト: DB 起動後 `make api-local` / `make batch-local`
#
# permission denied (/var/run/docker.sock) のとき:
#   sudo usermod -aG docker $USER  # 再ログイン後は sudo 不要
#   または: make DOCKER="sudo docker" up
DOCKER ?= docker

.PHONY: up db down batch mailworker logs health api-local batch-local mailworker-local

up:
	$(DOCKER) compose up -d --build db api

# API をホストで動かすときは db のみ起動し、ポート 8080 の競合を避ける。
db:
	$(DOCKER) compose up -d --build db

down:
	$(DOCKER) compose down

batch:
	$(DOCKER) compose --profile tools run --rm batch

mailworker:
	$(DOCKER) compose --profile tools run --rm mailworker

api-local:
	./scripts/run-api-local.sh

batch-local:
	./scripts/run-batch-local.sh

mailworker-local:
	./scripts/run-mailworker-local.sh

logs:
	$(DOCKER) compose logs -f api

health:
	curl -sf http://localhost:8080/health && echo OK
