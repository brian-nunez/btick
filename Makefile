.PHONY: templ api worker tailwind dev

templ:
	templ generate --watch --proxy="http://localhost:8080" --open-browser=false

api:
	air \
		--build.cmd "go build -o tmp/bin/scheduler-api ./cmd/scheduler-api" \
		--build.bin "tmp/bin/scheduler-api" \
		--build.delay "100" \
		--build.exclude_dir "node_modules" \
		--build.include_ext "go" \
		--build.stop_on_error "false" \
		--misc.clean_on_exit true

worker:
	air \
		--build.cmd "go build -o tmp/bin/scheduler-worker ./cmd/scheduler-worker" \
		--build.bin "tmp/bin/scheduler-worker" \
		--build.delay "100" \
		--build.exclude_dir "node_modules" \
		--build.include_ext "go" \
		--build.stop_on_error "false" \
		--misc.clean_on_exit true

tailwind:
	tailwindcss -i ./assets/css/input.css -o ./assets/css/output.css --watch

dev:
	make -j3 templ tailwind api
