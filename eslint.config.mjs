import js from '@eslint/js';
import tseslint from 'typescript-eslint';
import angular from 'angular-eslint';
import flyspace from '@flyspace/eslint-plugin';

// Flat config (ESLint v9). TypeScript + Angular rules on .ts; Angular's inline templates are
// extracted by processInlineTemplates and linted as HTML. The FlySpace design-token rule is the
// build-time half of the styling contract — it blocks hardcoded colors and lengths in component
// styles, so only var(--fly-*) tokens get through. CSS/SCSS files are covered by
// @flyspace/stylelint-config (the `lint` script runs stylelint alongside this).
export default tseslint.config(
  {
    ignores: ['dist/**', '.angular/**', 'coverage/**', 'public/**', 'scripts/**', '**/index.html', '**/*.config.{js,mjs,cjs,ts,mts}'],
  },
  {
    files: ['**/*.ts'],
    extends: [
      js.configs.recommended,
      ...tseslint.configs.recommended,
      ...angular.configs.tsRecommended,
      flyspace.configs.recommended,
    ],
    processor: angular.processInlineTemplates,
    rules: {
      '@angular-eslint/component-selector': ['error', { type: 'element', prefix: 'fly', style: 'kebab-case' }],
      '@angular-eslint/directive-selector': ['error', { type: 'attribute', prefix: 'fly', style: 'camelCase' }],
    },
  },
  {
    files: ['**/*.html'],
    extends: [...angular.configs.templateRecommended, ...angular.configs.templateAccessibility],
  },
);
