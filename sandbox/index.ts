import { WasmRouter } from "../router.ts";
import { Hono } from "hono";

const app = new Hono({
  router: new WasmRouter(),
});

app.get("/", (c) => {
  return c.text("Hello, World!");
});

export default app;
