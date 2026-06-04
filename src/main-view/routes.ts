import { Routes } from '@angular/router';

import { CalendarApi } from '../data/calendar-api';
import { MainViewComponent } from './main-view.component';

/**
 * The extension's routed surface, exposed via Native Federation as `./routes`. The shell mounts it
 * under `/apps/<id>` with a scoped FLYSPACE_SDK on the parent route; CalendarApi is provided here so
 * it resolves that scoped SDK.
 */
export const routes: Routes = [
  {
    path: '',
    providers: [CalendarApi],
    component: MainViewComponent,
  },
];
