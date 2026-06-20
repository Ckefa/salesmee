const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

const JS_DIR = path.resolve(__dirname, '..', 'web/static/js');
const OUT_DIR = path.resolve(JS_DIR, 'bundles');
const VENDOR_DIR = path.resolve(JS_DIR, 'vendor');

const bundles = {
  'business-bundle': [
    'modules/shared.js',
    'modules/ws.js',
    'modules/chat_common.js',
    'modules/business.js',
    'modules/business_chat.js',
    'core/app.js',
    'modules/assist.js',
    'modules/wizard_common.js',
    'modules/product_picker.js',
    'modules/service_picker.js',
    'modules/onboarding.js',
  ],
  'client-bundle': [
    'modules/shared.js',
    'modules/ws.js',
    'modules/chat_common.js',
    'modules/client.js',
    'modules/client_chat.js',
    'core/app.js',
    'modules/assist.js',
    'modules/wizard_common.js',
    'modules/product_picker.js',
    'modules/service_picker.js',
  ],
};

function build() {
  if (!fs.existsSync(OUT_DIR)) {
    fs.mkdirSync(OUT_DIR, { recursive: true });
  }

  for (const [name, files] of Object.entries(bundles)) {
    const parts = files.map(f => {
      const fullPath = path.resolve(JS_DIR, f);
      if (!fs.existsSync(fullPath)) {
        console.error(`Missing: ${fullPath}`);
        return '';
      }
      return fs.readFileSync(fullPath, 'utf8');
    });

    const combined = parts.join('\n');

    const tmpFile = path.resolve(OUT_DIR, `_${name}.js`);
    fs.writeFileSync(tmpFile, combined, 'utf8');

    const outFile = path.resolve(OUT_DIR, `${name}.js`);
    try {
      execSync(`npx esbuild "${tmpFile}" --minify --outfile="${outFile}"`, {
        stdio: 'inherit',
        cwd: path.resolve(__dirname, '..'),
      });
      const outSize = fs.statSync(outFile).size;
      console.log(`✓ ${name}.js (${(outSize / 1024).toFixed(1)} KB)`);
    } catch (err) {
      console.error(`✗ ${name}.js build failed:`, err.message);
    } finally {
      fs.unlinkSync(tmpFile);
    }
  }
  console.log('Done.');
}

if (process.argv.includes('--watch')) {
  console.log('Watching for changes...');
  const watchDirs = [path.resolve(JS_DIR, 'modules'), path.resolve(JS_DIR, 'core')];
  watchDirs.forEach(dir => {
    fs.watch(dir, { recursive: false }, (event, filename) => {
      if (filename && filename.endsWith('.js')) {
        console.log(`Change detected: ${filename}`);
        build();
      }
    });
  });
} else {
  build();
}
