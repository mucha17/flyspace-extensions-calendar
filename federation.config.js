// Native Federation config. Produces remoteEntry.json + the federated bundles at `ng build`.
const { withNativeFederation, shareAll } = require('@angular-architects/native-federation/config');

module.exports = withNativeFederation({
  name: 'calendar',

  exposes: {
    // Routed surface: the shell mounts these under /apps/<id> via loadChildren (convention).
    './routes': './src/main-view/routes.ts',
    // Named components: mounted in place by the shell's ExtensionMountComponent.
    './MainView': './src/main-view/main-view.component.ts',
    './DashboardWidget': './src/dashboard-widget/dashboard-widget.component.ts',
    './SettingsPanel': './src/settings-panel/settings-panel.component.ts',
  },

  shared: {
    ...shareAll({ singleton: true, strictVersion: true, requiredVersion: 'auto' }),
  },

  skip: [
    // packages not to federate (usually internal-only)
  ],
});
