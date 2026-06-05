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
    // singleton so the extension reuses the shell's Angular/rxjs/SDK; strictVersion: false so a
    // patch/minor skew between the shell's and the extension's build does not load a second copy
    // (the cause of NG0203). Only the major must match — enforced by the manifest peerRuntime.
    ...shareAll({ singleton: true, strictVersion: false, requiredVersion: 'auto' }),
  },

  skip: [
    // packages not to federate (usually internal-only)
  ],
});
