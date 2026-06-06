import { defineConfig } from 'vitest/config';

export default defineConfig({
  test: {
    globals: false,
    include: ['src/**/*.spec.ts'],
    setupFiles: ['src/test-setup.ts'],
    // @flyspace/* and @angular/* ship extensionless ESM relative imports that Vitest's default
    // resolver can't follow from node_modules. Inlining routes them through Vite's pipeline, which
    // resolves the extensions — needed for any test that touches the SDK token or Angular DI.
    server: {
      deps: {
        inline: [/@flyspace\//, /@angular\//],
      },
    },
    coverage: {
      provider: 'v8',
      reporter: ['text', 'html', 'lcov'],
      include: ['src/**/*.ts'],
      // Dev-only harness, federation/dev entrypoints, and lazy route wiring carry no shippable
      // logic to cover.
      exclude: [
        'src/**/*.spec.ts',
        'src/playground/**',
        'src/main.ts',
        'src/bootstrap.ts',
        'src/test-setup.ts',
        'src/**/routes.ts',
      ],
      // Floors set just under current coverage to catch regressions without churn.
      thresholds: {
        statements: 90,
        branches: 80,
        functions: 90,
        lines: 90,
      },
    },
  },
});
