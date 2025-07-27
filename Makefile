PORT=8080


all: free-port
	go run ./app


free-port:
	@echo "Освобождаем порт $(PORT)..."
	@kill -9 $$(lsof -ti :$(PORT)) 2>/dev/null || true