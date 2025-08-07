# Real-Time Trading System

A high-performance, modular trading engine built with Go, following hexagonal architecture and domain-driven design principles.

## Architecture

- **Hexagonal Architecture** - Clean separation of business logic and external dependencies
- **Domain-Driven Design** - Rich domain models with clear ubiquitous language
- **Event-Driven** - Loose coupling through event bus and streams
- **Test-Driven Development** - Comprehensive test coverage with TDD methodology

## Quick Start

```bash
# Setup development environment
make dev-setup

# Run tests
make test

# Build and run
make build
make run
```

## Development Workflow

1. **Research Phase** - Study best practices before implementation
2. **Feature Branch** - Create branch for each feature/todo item
3. **TDD Cycle** - Write tests first, implement, refactor
4. **Quality Gates** - Pass lint, vet, security, and performance checks
5. **Staging Merge** - Merge to staging after all tests pass

## Project Structure

```
├── cmd/                     # Application entry points
├── internal/                # Private application code
│   ├── adapters/           # External system adapters
│   ├── core/               # Business logic (Domain layer)
│   ├── data/               # Data processing pipeline
│   ├── ml/                 # Machine learning components
│   └── strategies/         # Trading strategies
├── pkg/                    # Public packages
├── tests/                  # Integration and E2E tests
└── docs/                   # Documentation
```

## Performance Requirements

- Order execution: <10ms p99
- Risk calculation: <1ms p99  
- ML inference: <50ms p99
- Unit test coverage: ≥95%
- Integration test coverage: ≥80%

## Components

- **Core Domain** - Assets, orders, positions, portfolio
- **Risk Management** - Position sizing, drawdown monitoring, limits
- **Execution Engine** - Order management with C++ optimization
- **ML Pipeline** - Feature engineering, model inference, drift detection
- **Data Pipeline** - Real-time ingestion, preprocessing, storage
- **Trading Strategies** - Mean reversion, momentum, arbitrage, ML-based
- **Backtesting** - Historical simulation and performance analysis