# Makefile for Trading Engine C++ Execution Engine
# Optimized for high-performance, low-latency trading

CXX = g++
AR = ar

# Compiler flags for maximum performance
CXXFLAGS = -std=c++17 -Wall -Wextra -pedantic
CXXFLAGS += -O3 -march=native -mtune=native
CXXFLAGS += -ffast-math -funroll-loops -finline-functions
CXXFLAGS += -pthread

# Debug flags (use with DEBUG=1)
DEBUG_FLAGS = -g3 -O0 -DDEBUG -fsanitize=address,undefined

# Include directories
INCLUDES = -I. -I../..

# Libraries
LIBS = -lpthread -lrt -ldl
NUMA_LIBS = -lnuma

# Source files
SOURCES = order_engine.cpp
OBJECTS = $(SOURCES:.cpp=.o)

# Targets
TARGET_LIB = liborder_engine.a
TARGET_SO = liborder_engine.so
TARGET_TEST = test_engine

# Default target
all: $(TARGET_LIB) $(TARGET_SO)

# Debug build
debug: CXXFLAGS = $(DEBUG_FLAGS) -std=c++17 -Wall -Wextra -pthread $(INCLUDES)
debug: clean all

# Static library
$(TARGET_LIB): $(OBJECTS)
	$(AR) rcs $@ $^

# Shared library
$(TARGET_SO): $(OBJECTS)
	$(CXX) -shared -fPIC -o $@ $^ $(LIBS)

# Object files
%.o: %.cpp order_engine.h
	$(CXX) $(CXXFLAGS) $(INCLUDES) -fPIC -c $< -o $@

# Test executable
$(TARGET_TEST): test_engine.cpp $(TARGET_LIB)
	$(CXX) $(CXXFLAGS) $(INCLUDES) -o $@ $< $(TARGET_LIB) $(LIBS)

# Test target
test: $(TARGET_TEST)
	./$(TARGET_TEST)

# Check dependencies
check-deps:
	@echo "Checking dependencies..."
	@ldconfig -p | grep -q libnuma && echo "✓ libnuma found" || echo "ℹ libnuma missing (install libnuma-dev for NUMA optimization)"
	@which g++ > /dev/null && echo "✓ g++ found" || echo "✗ g++ missing"
	@which ar > /dev/null && echo "✓ ar found" || echo "✗ ar missing"

# Clean build artifacts
clean:
	rm -f $(OBJECTS) $(TARGET_LIB) $(TARGET_SO) $(TARGET_TEST)
	rm -f *.gcda *.gcno gmon.out
	rm -f core core.*

# Show compiler info
info:
	@echo "Compiler: $(CXX)"
	@$(CXX) --version
	@echo "Flags: $(CXXFLAGS)"

# Install system-wide
install: $(TARGET_LIB) $(TARGET_SO)
	sudo mkdir -p /usr/local/include/trading
	sudo mkdir -p /usr/local/lib
	sudo cp order_engine.h /usr/local/include/trading/
	sudo cp $(TARGET_LIB) /usr/local/lib/
	sudo cp $(TARGET_SO) /usr/local/lib/
	sudo ldconfig

# Help
help:
	@echo "Trading Engine C++ Execution Engine Makefile"
	@echo "Targets:"
	@echo "  all        - Build static and shared libraries (default)"
	@echo "  test       - Build and run test executable"
	@echo "  debug      - Build with debug flags"
	@echo "  check-deps - Check for required dependencies"
	@echo "  clean      - Remove build artifacts"
	@echo "  install    - Install headers and libraries system-wide"
	@echo "  info       - Show compiler information"
	@echo "  help       - Show this help message"

.PHONY: all test debug check-deps clean install info help