GOTEST = go test
BENCHMARK_DIR = benchmarks
PACKAGE_PATH = ./internal/handler/
BROWSER_PATH = '/mnt/c/Program Files/Mozilla Firefox/firefox.exe' # Здесь указываем путь к своему браузеру
SHORTENER_PATH = ./cmd/shortener/bin/urlshrt
BIN_DIR = ./cmd/shortener/bin

# Цель по умолчанию
all: benchmark-memory benchmark-cpu pprof

# Цель для создания папки бинарника приложения
$(BIN_DIR):
	mkdir -p $(BIN_DIR)

# Цель для создания директории benchmarks
$(BENCHMARK_DIR):
	mkdir -p $(BENCHMARK_DIR)

# Цель для выполнения бенчмарков памяти
benchmark-memory: $(BENCHMARK_DIR)
	go mod download
	$(GOTEST) -bench . ./internal/handler/ -benchmem | grep -E '/op|PASS|ok |FAIL' > $(BENCHMARK_DIR)/benchmarks-memory.md

# Цель для выполнения бенчмарков CPU
# Возможно, придется немного подождать. Если выполняется прям очень долго - поменяйте count или просто уберите его
benchmark-cpu: $(BENCHMARK_DIR)
	go mod download
	$(GOTEST) -bench=. -count=3 -cpuprofile=$(BENCHMARK_DIR)/cpu.out $(PACKAGE_PATH) | grep -E '/op|PASS|ok |FAIL' > $(BENCHMARK_DIR)/benchmarks-cpu.md

# Цель для вывода в браузер профиля CPU
pprof: benchmark-cpu
	BROWSER=$(BROWSER_PATH) go tool pprof -http :8080 $(BENCHMARK_DIR)/cpu.out

# Цель для запуска приложения
shortener: $(BIN_DIR)
	go mod download
	go build -o $(BIN_DIR)/urlshrt ./cmd/shortener/
	docker-compose up -d
	$(SHORTENER_PATH) -d "host=localhost dbname=urlshrt user=urlshrt password=urlshrt port=3000 sslmode=disable"

shortener-secure: $(BIN_DIR)
	openssl req -x509 -nodes -days 365 -newkey rsa:2048 -keyout cert/localhost.key -out cert/localhost.crt -subj "/C=RU/ST=Moscow/L=Moscow/O=Localhost/OU=Dev/CN=localhost"
	go mod download
	go build -o $(BIN_DIR)/urlshrt ./cmd/shortener/
	docker-compose up -d
	$(SHORTENER_PATH) -d "host=localhost dbname=urlshrt user=urlshrt password=urlshrt port=3000 sslmode=disable" -s -e

# Цель для выполнения всех тестов
test:
	go mod download
	go test ./... -cover -count=1

# goimports надо установить до выполнения этой цели
goimports:
	goimports -local "github.com/PoorMercymain/urlshrt" -w .

# Цель для очистки
clean:
	rm -f $(BENCHMARK_DIR)/benchmarks-memory.md
	rm -f $(BENCHMARK_DIR)/benchmarks-cpu.md $(BENCHMARK_DIR)/cpu.out
	rm -f $(SHORTENER_PATH)