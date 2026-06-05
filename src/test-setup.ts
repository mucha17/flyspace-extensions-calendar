// Angular's @flyspace-adjacent dependencies (@angular/forms, @angular/common) ship partially
// compiled (`ngDeclare`) and fall back to JIT compilation at runtime. Loading the compiler here,
// before any spec imports a component, registers that JIT facade so instantiating components in
// tests works without the full Angular build pipeline.
import '@angular/compiler';
