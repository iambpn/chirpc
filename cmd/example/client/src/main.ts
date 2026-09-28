// This script walks through the example task API with a fully typed client.
// Run the server first, then generate the schema with "pnpm gen" and run "pnpm start".
import axios, { isAxiosError } from "axios";
import { TypedAxios, type TypedAxiosError } from "ts-axios-wrapper";
import type { ApiSchema } from "./apiSchema.js";

// Named types can be read from the schema, so the client never repeats a Go struct.
type Task = ApiSchema["GET"]["/api/v1/tasks/:taskId"]["response"];
type APIError = ApiSchema["ERROR_HANDLER"]["/"]["response"];

// ts-axios-wrapper is published as CommonJS, so TypeScript reads its axios types from a
// different file than the ESM axios types used here. Both describe the same instance.
type WrapperAxiosInstance = NonNullable<ConstructorParameters<typeof TypedAxios>[0]>;

const baseURL = process.env.API_URL ?? "http://localhost:8080";

// Makes a client for one user. "indexes: null" sends arrays as ?tag=a&tag=b,
// which is the form the Go query decoder reads.
function clientFor(token?: string): TypedAxios<ApiSchema> {
  const instance = axios.create({
    baseURL,
    headers: token ? { Authorization: `Bearer ${token}` } : {},
    paramsSerializer: { indexes: null },
  });
  return new TypedAxios<ApiSchema>(instance as unknown as WrapperAxiosInstance);
}

// Returns the typed error body when the request fails with an API error.
async function expectError(request: Promise<unknown>): Promise<APIError> {
  try {
    await request;
  } catch (error) {
    if (isAxiosError(error) && error.response) {
      return (error as TypedAxiosError<ApiSchema>).response!.data;
    }
    throw error;
  }
  throw new Error("The request should have failed, but it succeeded.");
}

function showTask(task: Task): string {
  const done = task.checklist.filter((item) => item.done).length;
  return `${task.id} [${task.status}/${task.priority}] ${task.title} (${done}/${task.checklist.length} steps, tags: ${task.tags.join(", ") || "none"})`;
}

async function main(): Promise<void> {
  const alice = clientFor("alice-token");
  const bob = clientFor("bob-token");

  const health = await clientFor().GET("/health", {});
  console.log(`1. Health: ${health.status} at ${health.time}`);

  const me = await alice.GET("/api/v1/account/me", {});
  console.log(`2. Signed in as ${me.name} <${me.email}>`);

  // The body type comes from CreateTaskBody. Leaving out "title" is a compile error.
  const created = await alice.POST("/api/v1/tasks", {
    body: {
      title: "Try the chirpc example",
      priority: "high",
      tags: ["demo", "docs"],
      dueAt: new Date(Date.now() + 24 * 60 * 60 * 1000).toISOString(),
      checklist: [{ text: "Start the server" }, { text: "Run the client", done: true }],
    },
  });
  console.log(`3. Created ${showTask(created)}`);

  // Query parameters are typed too. "status" only accepts "todo", "doing", or "done".
  const page = await alice.GET("/api/v1/tasks", {
    query: { status: "todo", tag: ["demo"], limit: 10 },
  });
  console.log(`4. Found ${page.total} todo task(s) tagged "demo":`);
  for (const task of page.items) {
    console.log(`   - ${showTask(task)}`);
  }

  // Path parameters are filled in from "params".
  const updated = await alice.PATCH("/api/v1/tasks/:taskId", {
    params: { taskId: created.id },
    body: { status: "doing", tags: ["demo"] },
  });
  console.log(`5. Updated ${showTask(updated)}`);

  // Bob cannot see Alice's task, so the server answers 404.
  const hidden = await expectError(bob.GET("/api/v1/tasks/:taskId", { params: { taskId: created.id } }));
  console.log(`6. Bob gets ${hidden.status}: ${hidden.message}`);

  // chirpc checks required fields at every level before the handler runs.
  // The cast skips the compile-time check on purpose, to show the server check.
  const invalid = await expectError(
    alice.POST("/api/v1/tasks", {
      body: { title: "Bad checklist", checklist: [{ done: true }] } as unknown as { title: string },
    }),
  );
  console.log(`7. Invalid body gets ${invalid.status}: ${invalid.message}`);
  for (const [field, messages] of Object.entries(invalid.fields ?? {})) {
    console.log(`   - ${field}: ${messages.join(" ")}`);
  }

  await alice.DELETE("/api/v1/tasks/:taskId", { params: { taskId: created.id } });
  const gone = await expectError(alice.GET("/api/v1/tasks/:taskId", { params: { taskId: created.id } }));
  console.log(`8. After delete, the task gets ${gone.status}: ${gone.message}`);
}

main().catch((error: unknown) => {
  if (isAxiosError(error)) {
    console.error(`The request failed: ${error.message}`, error.response?.data ?? "");
  } else {
    console.error(error);
  }
  process.exitCode = 1;
});
