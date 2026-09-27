import { TypedAxios } from "ts-axios-wrapper";
import type { ApiSchema } from "../../../apiSchema.js";

export const api = new TypedAxios<ApiSchema>();

api.request("GET", "/", {
  query: {
    name: "John Doe",
  },
  body: {
    name: "John Doe",
  },
});

api.GET("/", {
  query: {
    name: "John Doe",
  },
  body: {
    age: 25,
    name: "John Doe",
  },
});

api.GET("/:test", {
  params: { test: "example" },
});

api.POST("/users", {
  body: { name: "Ada" },
});
