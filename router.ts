import { METHOD_NAME_ALL_LOWERCASE, METHODS, type Result, type Router } from "hono/router";
import { loadWasmRouter } from "./wasm/router.ts";

export class WasmRouter<T> implements Router<T> {
  name: string = "WasmRouter";
  // deno-lint-ignore no-explicit-any
  wasmRouter?: Record<string, any>;
  mehotds: string[] = [METHOD_NAME_ALL_LOWERCASE, ...METHODS];
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
    if (!this.mehotds.includes(method.toLowerCase())) {
      this.mehotds.push(method.toLowerCase());
    }
    this.routes.push(handler);
    this.wasmRouter.Add(this.mehotds.indexOf(method.toLowerCase()), path, this.routes.length - 1);
  }

  match(method: string, path: string): Result<T> {
    const matchResult = this.wasmRouter?.Match(method, path);
    console.log(matchResult)
    return matchResult;
  }
}
