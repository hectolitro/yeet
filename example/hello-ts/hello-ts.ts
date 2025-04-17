/**
 * Copyright 2025 AUTHORS
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { serve } from "https://deno.land/std/http/server.ts";

const handler = (req: Request): Response => {
  return new Response("Hello World from Deno!", {
    status: 200,
    headers: { "content-type": "text/plain" },
  });
};

console.log("Listening on http://localhost:8081");
await serve(handler, { port: 8081 })
