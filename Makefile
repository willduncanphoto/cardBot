.PHONY: build test clean qa-050 qa-051-sleepwake qa-051-permissions

build:
	go build -ldflags="-s -w" -o cardbot .

test:
	go test ./... -count=1 -race

qa-050:
	./scripts/qa_050_smoke.sh

qa-051-sleepwake:
	./scripts/qa_051_sleepwake_capture.sh

qa-051-permissions:
	./scripts/qa_051_permissions_capture.sh

clean:
	rm -f cardbot coverage.out
