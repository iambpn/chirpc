import { TypedAxios } from "ts-axios-wrapper";
import type { ApiSchema } from "../../apiSchema.js";

export const api = new TypedAxios<ApiSchema>();

api.request("GET", "/", {
  body: {
    name: "John Doe",
  },
});

api.GET("/", {
  body: {
    age: 25,
    name: "John Doe",
  },
});

api.GET("/:test", {
  params: { test: "example" },
});
