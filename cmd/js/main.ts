import { TypedAxios } from "ts-axios-wrapper";
import type { ApiSchema } from "../../apiSchema.js";

export const api = new TypedAxios<ApiSchema>();

api.request("GET", "/", {});

api.GET("/", {});
