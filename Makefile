build: deps
	mkdir -p build
	cd backend; go build -o ../build/mcmaster-backend .
	cd frontend; yarn build

deps:
	cd frontend; yarn install

run-dev:
	./lib/start_dev.sh
