// Computes sha384(remoteEntry) and updates flyspace-extension.json integrity.bundleHash.
// TODO: implement properly. Sketch:
//
//   const buf = readFileSync('dist/template/remoteEntry.json');
//   const hash = 'sha384-' + createHash('sha384').update(buf).digest('base64');
//   const manifest = JSON.parse(readFileSync('flyspace-extension.json', 'utf8'));
//   manifest.integrity.bundleHash = hash;
//   writeFileSync('flyspace-extension.json', JSON.stringify(manifest, null, 2));
console.log('TODO: compute integrity hash');
