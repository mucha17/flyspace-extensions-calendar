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
  },
});
