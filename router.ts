import type { Result, Router } from "hono/router";
import { loadWasmRouter } from "./wasm/router.ts";

export class WasmRouter<T> implements Router<T> {
  name: string = "WasmRouter";
  // deno-lint-ignore no-explicit-any
  wasmRouter?: Record<string, any>;
  routes: T[] = [];

  constructor() {
    // @ts-expect-error: Async Constructor
    return new Promise<WasmRouter<T>>((resolve, reject) => {
      loadWasmRouter()
        .then((wasmRouter) => {
          this.wasmRouter = wasmRouter.exports;
          resolve(this);
        })
        .catch((err) => {
          reject(err);
        })
    })
  }

  add(method: string, path: string, handler: T) {
    if (!this.wasmRouter) {
      throw new Error("You should use `await new WasmRouter()`");
    }
    this.routes.push(handler);
    this.wasmRouter.Add(method, path, this.routes.length - 1);
  }

  match(method: string, path: string): Result<T> {
    const matchResult = this.wasmRouter?.Match(method, path);
    console.log(matchResult)
    return matchResult;
  }
}
