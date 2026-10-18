import PocketBase from "pocketbase";

export const pb = new PocketBase(
  typeof window !== "undefined" ? window.location.origin : "http://127.0.0.1:8090"
);
