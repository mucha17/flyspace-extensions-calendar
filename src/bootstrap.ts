import { bootstrapApplication } from '@angular/platform-browser';
import { provideRouter } from '@angular/router';
import { FLYSPACE_SDK } from '@flyspace/sdk';
import { PlaygroundComponent } from './playground/playground.component';
import { stubSdk } from './playground/stub-sdk';
import { routes } from './main-view/routes';

// Standalone dev bootstrap. Inside the shell this file is NOT used — the shell loads the exposed
// `./routes` / components and provides the real scoped FLYSPACE_SDK itself.
bootstrapApplication(PlaygroundComponent, {
  providers: [provideRouter(routes), { provide: FLYSPACE_SDK, useValue: stubSdk }],
}).catch((err) => console.error(err));
