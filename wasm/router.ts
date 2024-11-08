import * as fs from "node:fs";
import "./wasm_exec.js";

export function loadWasmRouter(): WebAssembly.Instance {
    // @ts-expect-error: Global
  const go = new Go();
  const instance = new WebAssembly.Instance(
    new WebAssembly.Module(
      // WILL SUPPORT BROWSER
      fs.readFileSync(new URL("./router.wasm", import.meta.url)),
    ),
    go.importObject,
  );
  go.run(instance);
  return instance;
}
