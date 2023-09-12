GOTEST = go test
BENCHMARK_DIR = benchmarks
PACKAGE_PATH = ./internal/handler/
BROWSER_PATH = '/mnt/c/Program Files/Mozilla Firefox/firefox.exe' # Здесь указываем путь к своему браузеру

# Цель по умолчанию
all: benchmark-memory benchmark-cpu pprof

# Цель для создания директории benchmarks
$(BENCHMARK_DIR):
	mkdir -p $(BENCHMARK_DIR)

# Цель для выполнения бенчмарков памяти
benchmark-memory: $(BENCHMARK_DIR)
	$(GOTEST) -bench . ./internal/handler/ -benchmem | grep -E '/op|PASS|ok |FAIL' > $(BENCHMARK_DIR)/benchmarks-memory.md

# Цель для выполнения бенчмарков CPU
# Возможно, придется немного подождать. Если выполняется прям очень долго - поменяйте count или просто уберите его
benchmark-cpu: $(BENCHMARK_DIR)
	$(GOTEST) -bench=. -count=3 -cpuprofile=$(BENCHMARK_DIR)/cpu.out $(PACKAGE_PATH) | grep -E '/op|PASS|ok |FAIL' > $(BENCHMARK_DIR)/benchmarks-cpu.md

pprof: benchmark-cpu
	BROWSER=$(BROWSER_PATH) go tool pprof -http :8080 $(BENCHMARK_DIR)/cpu.out

# Цель для очистки
clean:
	rm -f $(BENCHMARK_DIR)/benchmarks-memory.md
	rm -f $(BENCHMARK_DIR)/benchmarks-cpu.md $(BENCHMARK_DIR)/cpu.out