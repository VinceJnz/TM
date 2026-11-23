await import('file:///usr/local/go/lib/wasm/wasm_exec.js');

const [binary] = Deno.args;
if (!binary) throw new Error('Expected a Go WASM test binary');
const go = new Go();
go.argv = Deno.args;
go.env = Deno.env.toObject();
go.exit = (code) => Deno.exit(code);
const {instance} = await WebAssembly.instantiate(await Deno.readFile(binary), go.importObject);
await go.run(instance);
