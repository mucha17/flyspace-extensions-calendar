// Runs `ng build` and returns once the bundle is written. The esbuild-based builder in this setup
// finishes the build but does not exit the process; this wrapper resolves on the completion marker,
// terminates the child, and exits 0 so `pnpm build` / `make release` don't hang. A non-zero exit
// before the marker (a real build failure) is propagated.
import { spawn } from 'node:child_process';

const MARKER = 'bundle generation complete';

const child = spawn('pnpm', ['exec', 'ng', 'build'], { stdio: ['inherit', 'pipe', 'inherit'] });

let completed = false;
let seen = '';

child.stdout.on('data', (chunk) => {
  const text = chunk.toString();
  process.stdout.write(text);
  seen += text;
  if (!completed && seen.includes(MARKER)) {
    completed = true;
    child.kill('SIGTERM');
    // If esbuild ignores SIGTERM, force it shortly after.
    setTimeout(() => child.kill('SIGKILL'), 2000).unref();
  }
});

child.on('exit', (code, signal) => {
  if (completed) process.exit(0); // build finished; we terminated the lingering process
  process.exit(code ?? (signal ? 1 : 0));
});

child.on('error', (err) => {
  console.error('build-bundle: failed to start ng build:', err.message);
  process.exit(1);
});
