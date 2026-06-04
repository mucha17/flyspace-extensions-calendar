// Native Federation entry. initFederation() sets up the import map for shared singletons; the
// standalone dev app then bootstraps. When consumed by the shell, only the federated `exposes`
// (built into remoteEntry.json) are used — this entry runs only for `ng serve` / the playground.
import { initFederation } from '@angular-architects/native-federation';

initFederation()
  .catch((err) => console.error(err))
  .then(() => import('./bootstrap'))
  .catch((err) => console.error(err));
