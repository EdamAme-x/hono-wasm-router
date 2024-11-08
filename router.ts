import type { Params, Result, Router } from 'hono/router'
import { METHOD_NAME_ALL } from 'hono/router'

export class WasmRouter<T> implements Router<T> {
  name: string = 'WasmRouter'

  add(method: string, path: string, handler: T) {

  }

  match(method: string, path: string): Result<T> {
    return [[]]
  }
}