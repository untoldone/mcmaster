build: deps
	cd backend; go build .
	cd frontend; yarn build

deps:
	cd frontend; yarn install

run-dev:
	./build/start_dev.sh
