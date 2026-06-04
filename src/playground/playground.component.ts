import { Component } from '@angular/core';
import { RouterOutlet } from '@angular/router';

/**
 * Dev playground root (standalone `ng serve`). Mounts the extension's own routes in a realistic-ish
 * context so the author can iterate without the full shell.
 */
@Component({
  selector: 'fly-template-playground',
  standalone: true,
  imports: [RouterOutlet],
  template: `
    <p style="opacity:0.6">flyspace-ext dev playground</p>
    <router-outlet />
  `,
})
export class PlaygroundComponent {}
