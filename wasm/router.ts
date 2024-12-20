import * as fs from "node:fs";
import "./wasm_exec.js";

export async function loadWasmRouter(): Promise<WebAssembly.Instance> {
  // @ts-expect-error: Global
  const go = new Go();
  const instance = await WebAssembly.instantiate(
    (fs.readFileSync(new URL("./router.wasm", import.meta.url))),
    go.importObject,
  )
  go.run(instance.instance);
  // https://github.com/tinygo-org/tinygo/issues1140#issuecomment-671261465
  go.importObject.env["syscall/js.finalizeRef"] = ()=> {};
  return instance.instance;
}
