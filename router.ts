import type { Result, Router } from "hono/router";
import { loadWasmRouter } from "./wasm/router.ts";

export class WasmRouter<T> implements Router<T> {
  name: string = "WasmRouter";
  // deno-lint-ignore no-explicit-any
  wasmRouter: any;

  constructor() {
    this.wasmRouter = loadWasmRouter().exports;
  }

  add(method: string, path: string, handler: T) {
    this.wasmRouter.Add(method, path, handler);
  }

  match(method: string, path: string): Result<T> {
    const matchResult = this.wasmRouter.Match(method, path);
    console.log(matchResult)
    return matchResult;
  }
}
