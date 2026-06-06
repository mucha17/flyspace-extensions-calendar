# Quality gate for the calendar extension. Frontend runs from the repo root; the backend is a
# separate Go module under backend/. The backend's integration tests (store, end-to-end) sit behind
# the `integration` build tag and bring up Postgres via testcontainers, so they need a running
# Docker daemon. The default test/gate targets include them; the *-unit targets stay container-free.

.PHONY: gate lint test test-frontend test-backend test-backend-unit \
        coverage coverage-frontend coverage-backend

# Full quality gate: lint + both test suites (incl. integration). The release build
# (`pnpm build`, which also refreshes the integrity hash) is a separate step.
gate: lint test

lint:
	pnpm lint

test: test-frontend test-backend

test-frontend:
	pnpm test

# Build + vet + race tests including the integration-tagged suites (requires Docker).
test-backend:
	cd backend && go build ./... && go vet ./... && go test ./... -race -tags=integration

# Container-free pass: excludes the integration-tagged suites.
test-backend-unit:
	cd backend && go build ./... && go vet ./... && go test ./... -race

coverage: coverage-frontend coverage-backend

coverage-frontend:
	pnpm test:coverage

coverage-backend:
	cd backend && go test ./... -tags=integration -covermode=atomic -coverprofile=coverage.out
	cd backend && go tool cover -func=coverage.out | tail -1
