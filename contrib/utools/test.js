// Run the uTools adapter's binary lookup and query path without uTools.
//
//   node test.js /path/to/lingbox "ip 10.0.0.0/8"
//
// Prints the launcher JSON; exits non-zero when the query fails.

const path = require('path');

const bin = process.argv[2];
const query = process.argv[3] || 'uuid';

if (!bin) {
  console.error('usage: node test.js <lingbox-binary> [query]');
  process.exit(2);
}

process.env.LINGBOX_BIN = path.resolve(bin);

const { queryOnce } = require('./preload.js');

queryOnce(query).then((resp) => {
  process.stdout.write(JSON.stringify(resp, null, 2) + '\n');
  process.exit(resp.error ? 1 : 0);
});
